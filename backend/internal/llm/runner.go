package llm

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/google/uuid"
)

// runner executes LLM calls with retry, repair, and auditing.
type runner struct {
	provider Provider
	auditor  Auditor
	cfg      ProviderConfig
	now      func() time.Time
	logger   *slog.Logger
}

// RunnerOption customizes a runner at construction.
type RunnerOption func(*runner)

// WithRunnerLogger attaches a structured logger so parse/rejection diagnostics
// (empty content, JSON extraction, unmarshal, repair) are observable. The
// provider layer already logs request/response; this logger covers the
// text→struct boundary that previously produced opaque failures.
func WithRunnerLogger(l *slog.Logger) RunnerOption {
	return func(r *runner) {
		if l != nil {
			r.logger = l
		}
	}
}

// NewRunner builds a Runner that retries, audits, and repairs outputs.
func NewRunner(provider Provider, auditor Auditor, cfg ProviderConfig, opts ...RunnerOption) Runner {
	r := &runner{provider: provider, auditor: auditor, cfg: cfg, now: time.Now, logger: slog.Default()}
	for _, o := range opts {
		o(r)
	}
	return r
}

const repairLimit = 2

func (r *runner) GenerateStructured(ctx context.Context, req Request, out any) (*Run, error) {
	start := time.Now()
	hash := requestHash(req.SystemPrompt, req.UserPrompt, req.PromptVersion)

	run := &Run{
		ID:            uuid.NewString(),
		Purpose:       req.Purpose,
		Model:         r.cfg.Model,
		PromptVersion: req.PromptVersion,
		EntityType:    req.EntityType,
		EntityID:      req.EntityID,
		RequestHash:   hash,
	}

	attempts := 0
	var lastErr error
	var text string

	for attempt := 0; attempt <= repairLimit; attempt++ {
		attempts++
		var resp *Response
		var err error

		// retry transient provider errors with backoff
		for retry := 0; retry <= r.cfg.MaxRetries; retry++ {
			req.Attempt = retry + 1
			resp, err = r.provider.Chat(ctx, req)
			if err == nil {
				break
			}
			if !retryable(err) {
				break
			}
			if retry < r.cfg.MaxRetries {
				delay := r.cfg.RetryBaseDelay * time.Duration(1<<retry)
				if err := sleepCtx(ctx, delay); err != nil {
					return nil, err
				}
			}
		}

		if err != nil {
			lastErr = err
			break
		}

		text = resp.Text
		run.InputTokens += resp.InputTokens
		run.OutputTokens += resp.OutputTokens

		// Stage 1: provider returned nothing at all.
		if strings.TrimSpace(text) == "" {
			run.StructuredOutput = nil
			r.logOutputRejected(req, run, "empty_provider_content", "", text, attempt)
			lastErr = &ErrValidation{Purpose: req.Purpose, Err: fmt.Errorf("llm: empty provider content")}
			if attempt < repairLimit {
				req.UserPrompt = repairPrompt(req.UserPrompt, "<empty response>")
				req.PromptVersion = req.PromptVersion + ".repair"
				continue
			}
			break
		}

		// Stage 2: extract the JSON document from the provider text (markdown
		// fences and leading/trailing prose).
		raw := []byte(extractJSON(text))
		if len(strings.TrimSpace(string(raw))) == 0 {
			r.logOutputRejected(req, run, "json_extraction_failure", "", text, attempt)
			lastErr = &ErrValidation{Purpose: req.Purpose, Err: fmt.Errorf("llm: could not extract a JSON document from the response")}
			if attempt < repairLimit {
				req.UserPrompt = repairPrompt(req.UserPrompt, text)
				req.PromptVersion = req.PromptVersion + ".repair"
				continue
			}
			break
		}

		// Stage 3: unmarshal the extracted JSON into the requested Go value.
		if err := json.Unmarshal(raw, out); err != nil {
			// Schema/struct shape mismatches surface here as type errors
			// (e.g. a string field receiving an object). Unknown fields are
			// ignored by encoding/json, so a valid-but-wrong-shaped document is
			// detected later by the caller's structured validation.
			r.logOutputRejected(req, run, "unmarshal_failure", err.Error(), string(raw), attempt)
			lastErr = &ErrValidation{Purpose: req.Purpose, Err: err}
			// repair attempt: ask the model to fix its own malformed output
			if attempt < repairLimit {
				req.UserPrompt = repairPrompt(req.UserPrompt, text)
				req.PromptVersion = req.PromptVersion + ".repair"
				continue
			}
			break
		}

		// success
		run.LatencyMs = time.Since(start).Milliseconds()
		run.Status = "success"
		run.StructuredOutput = []byte(text)
		if r.auditor != nil {
			_ = r.auditor.Record(ctx, run)
		}
		return run, nil
	}

	run.LatencyMs = time.Since(start).Milliseconds()
	run.Status = "failed"
	if lastErr != nil {
		run.Error = lastErr.Error()
	}
	if r.auditor != nil {
		_ = r.auditor.Record(ctx, run)
	}
	return run, lastErr
}

func (r *runner) GenerateText(ctx context.Context, req Request) (*Run, error) {
	start := time.Now()

	run := &Run{
		ID:            uuid.NewString(),
		Purpose:       req.Purpose,
		Model:         r.cfg.Model,
		PromptVersion: req.PromptVersion,
		EntityType:    req.EntityType,
		EntityID:      req.EntityID,
		RequestHash:   requestHash(req.SystemPrompt, req.UserPrompt, req.PromptVersion),
	}

	var resp *Response
	var err error
	for retry := 0; retry <= r.cfg.MaxRetries; retry++ {
		req.Attempt = retry + 1
		resp, err = r.provider.Chat(ctx, req)
		if err == nil {
			break
		}
		if !retryable(err) {
			break
		}
		if retry < r.cfg.MaxRetries {
			delay := r.cfg.RetryBaseDelay * time.Duration(1<<retry)
			if err := sleepCtx(ctx, delay); err != nil {
				return nil, err
			}
		}
	}

	if err != nil {
		run.LatencyMs = time.Since(start).Milliseconds()
		run.Status = "failed"
		run.Error = err.Error()
		if r.auditor != nil {
			_ = r.auditor.Record(ctx, run)
		}
		return run, err
	}

	run.LatencyMs = time.Since(start).Milliseconds()
	run.Status = "success"
	run.InputTokens = resp.InputTokens
	run.OutputTokens = resp.OutputTokens
	run.StructuredOutput = []byte(resp.Text)
	if r.auditor != nil {
		_ = r.auditor.Record(ctx, run)
	}
	return run, nil
}

// extractJSON pulls the first JSON object or array from model text.
func extractJSON(text string) string {
	s := strings.TrimSpace(text)
	// strip markdown fences
	if strings.HasPrefix(s, "```") {
		if idx := strings.IndexByte(s, '\n'); idx >= 0 {
			s = s[idx+1:]
		}
		s = strings.TrimSuffix(strings.TrimSpace(s), "```")
	}
	return strings.TrimSpace(s)
}

func repairPrompt(original, malformed string) string {
	return fmt.Sprintf(
		"Your previous response was not valid JSON and was rejected.\n\n"+
			"Here is the malformed output:\n%s\n\n"+
			"Respond again with ONLY valid strict JSON matching the exact schema requested. "+
			"Do not repeat the original instructions you were given. Original request:\n\n%s",
		truncate(malformed, 2000), original,
	)
}

func requestHash(system, user, version string) string {
	sum := sha256.Sum256([]byte(system + "\n\x00\n" + user + "\n" + version))
	return hex.EncodeToString(sum[:])
}

// logOutputRejected emits a diagnostic for a provider response that could not
// be turned into the requested Go value. It distinguishes the failure stage
// (empty content, JSON extraction, unmarshal) and captures a truncated view of
// the raw provider text so the exact reason is never reduced to an opaque
// "empty output". Long prompts are never logged (only model output), and the
// truncation keeps logs bounded.
func (r *runner) logOutputRejected(req Request, run *Run, stage, detail, raw string, attempt int) {
	attrs := []any{
		"purpose", req.Purpose,
		"prompt_version", req.PromptVersion,
		"entity_type", req.EntityType,
		"entity_id", req.EntityID,
		"model", r.cfg.Model,
		"attempt", attempt,
		"stage", stage,
	}
	if detail != "" {
		attrs = append(attrs, "detail", detail)
	}
	if strings.TrimSpace(raw) != "" {
		attrs = append(attrs, "raw_output_excerpt", truncate(strings.TrimSpace(raw), 800))
	}
	r.logger.Warn("llm_output_rejected", attrs...)
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
