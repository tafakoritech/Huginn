package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"
)

// openAIProvider implements Provider against any OpenAI-compatible chat completions API.
type openAIProvider struct {
	cfg    ProviderConfig
	client *http.Client
	logger *slog.Logger

	// mu guards jsonSchemaUnsupported. Once the upstream rejects the
	// json_schema response_format (e.g. DeepSeek returns HTTP 400), we cache
	// that capability for the provider's lifetime so we stop wasting a request
	// on json_schema for every retry attempt.
	mu                    sync.Mutex
	jsonSchemaUnsupported bool
}

// ProviderOption customizes an OpenAI-compatible provider at construction.
type ProviderOption func(*openAIProvider)

// WithLogger attaches a structured logger for per-request LLM observability.
func WithLogger(l *slog.Logger) ProviderOption {
	return func(p *openAIProvider) {
		if l != nil {
			p.logger = l
		}
	}
}

// NewOpenAICompatibleProvider creates a provider for an OpenAI-compatible base
// URL. It returns an error when the transport cannot be built (e.g. a malformed
// LLM_SOCKS5_PROXY address) so configuration mistakes fail fast at startup.
func NewOpenAICompatibleProvider(cfg ProviderConfig, opts ...ProviderOption) (Provider, error) {
	client, err := newHTTPClient(cfg)
	if err != nil {
		return nil, err
	}
	p := &openAIProvider{
		cfg:    cfg,
		client: client,
		logger: slog.Default(),
	}
	for _, o := range opts {
		o(p)
	}
	return p, nil
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model          string          `json:"model"`
	Messages       []chatMessage   `json:"messages"`
	Temperature    *float64        `json:"temperature,omitempty"`
	MaxTokens      int             `json:"max_tokens,omitempty"`
	ResponseFormat *responseFormat `json:"response_format,omitempty"`
	// DeepSeek / OpenAI compatible
}

type responseFormat struct {
	Type       string      `json:"type"`
	JSONSchema *jsonSchema `json:"json_schema,omitempty"`
}

type jsonSchema struct {
	Name   string `json:"name"`
	Strict bool   `json:"strict"`
	Schema any    `json:"schema"`
}

type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
	} `json:"usage"`
}

func (p *openAIProvider) Chat(ctx context.Context, req Request) (*Response, error) {
	if p.cfg.BaseURL == "" {
		return nil, fmt.Errorf("llm provider: %w", ErrNotConfigured)
	}
	// Plain text generation needs no structured response format.
	if req.JSONSchema == nil {
		return p.chatOnce(ctx, req, "", nil)
	}

	// Structured requests request progressively more compatible response
	// formats. The ordered candidate list depends on the configured mode:
	//
	//   - StructuredModeJSONObject: only json_object (the prompt carries the
	//     exact JSON contract and the runner validates the parsed structure).
	//     Some providers (e.g. DeepSeek deepseek-v4-flash) reject json_schema
	//     with HTTP 400 "This response_format type is unavailable now", so this
	//     mode never wastes a request on a known-invalid format.
	//   - StructuredModeJSONSchema: json_schema first, then json_object, then
	//     no format.
	//   - StructuredModeAuto (default): json_schema first; once a 400 proves the
	//     upstream rejects it we cache that capability (see jsonSchemaUnsupported)
	//     and jump straight to json_object for the rest of the provider's
	//     lifetime.
	var formats []struct {
		typ    string
		schema *jsonSchema
	}
	mode := p.cfg.StructuredMode
	if mode == "" {
		mode = StructuredModeAuto
	}
	allowSchema := mode != StructuredModeJSONObject
	if allowSchema && (mode == StructuredModeJSONSchema || p.jsonSchemaOK()) {
		formats = append(formats, struct {
			typ    string
			schema *jsonSchema
		}{"json_schema", &jsonSchema{Name: "structured_output", Strict: true, Schema: req.JSONSchema}})
	}
	formats = append(formats,
		struct {
			typ    string
			schema *jsonSchema
		}{"json_object", nil},
		struct {
			typ    string
			schema *jsonSchema
		}{"", nil},
	)

	var lastErr error
	for _, f := range formats {
		resp, err := p.chatOnce(ctx, req, f.typ, f.schema)
		if err == nil {
			return resp, nil
		}
		lastErr = err
		if formatUnsupported(err) {
			// The upstream rejected the requested response_format; remember it
			// for the lifetime of this provider and move to the next format.
			p.markJSONSchemaUnsupported()
			continue
		}
		return nil, err
	}
	return nil, lastErr
}

// jsonSchemaOK reports whether the provider still believes the upstream accepts
// the json_schema response_format.
func (p *openAIProvider) jsonSchemaOK() bool {
	p.mu.Lock()
	defer p.mu.Unlock()
	return !p.jsonSchemaUnsupported
}

// markJSONSchemaUnsupported caches that the upstream rejects json_schema.
func (p *openAIProvider) markJSONSchemaUnsupported() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.jsonSchemaUnsupported = true
}

func (p *openAIProvider) chatOnce(ctx context.Context, req Request, formatType string, schema *jsonSchema) (*Response, error) {
	msgs := []chatMessage{
		{Role: "system", Content: req.SystemPrompt},
		{Role: "user", Content: req.UserPrompt},
	}

	body := chatRequest{
		Model:       p.cfg.Model,
		Messages:    msgs,
		Temperature: req.Temperature,
		MaxTokens:   req.MaxTokens,
	}

	switch formatType {
	case "json_schema":
		body.ResponseFormat = &responseFormat{Type: "json_schema", JSONSchema: schema}
	case "json_object":
		body.ResponseFormat = &responseFormat{Type: "json_object"}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("llm: marshal request: %w", err)
	}

	// Log the configured per-attempt request timeout and how much time remains
	// on the parent (job) deadline before this attempt. This makes it obvious
	// when an attempt was starved by a shared expiring context.
	remaining := int64(-1)
	if d, ok := ctx.Deadline(); ok {
		remaining = time.Until(d).Milliseconds()
	}
	logAttrs := []any{
		"purpose", req.Purpose,
		"model", p.cfg.Model,
		"format", formatLabel(formatType),
		"input_chars", len(payload),
		"attempt", req.Attempt,
		"request_timeout_ms", p.cfg.Timeout.Milliseconds(),
		"remaining_deadline_ms", remaining,
		"entity_type", req.EntityType,
		"entity_id", req.EntityID,
	}
	for k, v := range req.Meta {
		logAttrs = append(logAttrs, k, v)
	}
	p.logger.Info("llm_request_started", logAttrs...)

	url := strings.TrimRight(p.cfg.BaseURL, "/") + "/chat/completions"
	httpReq, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		p.logger.Error("llm_request_failed", append(logAttrs, "error", err.Error(), "stage", "http")...)
		return nil, fmt.Errorf("llm: build request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	// Never log the API key or the Authorization header.
	httpReq.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)

	// Each network attempt gets its own fresh, full request timeout decoupled
	// from the parent job context's wall-clock deadline. Parent cancellation
	// (job aborted / worker shutdown) still propagates.
	attemptCtx, cancel := attemptContext(ctx, p.cfg.Timeout)
	defer cancel()

	start := time.Now()
	resp, err := p.client.Do(httpReq.WithContext(attemptCtx))
	latency := time.Since(start)
	if err != nil {
		stage := networkStage(err)
		p.logger.Error("llm_request_failed", append(logAttrs,
			"error", err.Error(),
			"latency_ms", latency.Milliseconds(),
			"stage", stage,
		)...)
		return nil, &ErrProvider{Purpose: req.Purpose, Err: fmt.Errorf("llm: request: %w (stage: %s)", err, stage)}
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, 1<<22))
	if err != nil {
		p.logger.Error("llm_request_failed", append(logAttrs, "error", err.Error(), "latency_ms", latency.Milliseconds(), "stage", "http")...)
		return nil, &ErrProvider{Purpose: req.Purpose, Err: fmt.Errorf("llm: read body: %w", err)}
	}

	if resp.StatusCode != http.StatusOK {
		// Capture the provider's structured error (code + message) plus a
		// truncated body for diagnostics. Never log the API key, the
		// Authorization header, or JWTs — those are not part of the response
		// body and are never added to the log attributes.
		providerCode, providerMessage := parseProviderError(respBody)
		p.logger.Error("llm_request_failed", append(logAttrs,
			"status", resp.StatusCode,
			"provider_error_code", providerCode,
			"provider_error_message", providerMessage,
			"latency_ms", latency.Milliseconds(),
			"provider_error_body", truncate(string(respBody), 400),
		)...)
		return nil, &ErrProvider{
			Purpose:    req.Purpose,
			StatusCode: resp.StatusCode,
			Err:        fmt.Errorf("llm: upstream status %d (code=%s): %s", resp.StatusCode, providerCode, truncate(string(respBody), 500)),
		}
	}

	var cr chatResponse
	if err := json.Unmarshal(respBody, &cr); err != nil {
		p.logger.Error("llm_request_failed", append(logAttrs, "error", err.Error(), "latency_ms", latency.Milliseconds(), "stage", "parse")...)
		return nil, &ErrProvider{Purpose: req.Purpose, Err: fmt.Errorf("llm: decode response: %w", err)}
	}
	if len(cr.Choices) == 0 {
		p.logger.Error("llm_request_failed", append(logAttrs, "error", "empty choices", "latency_ms", latency.Milliseconds(), "stage", "parse")...)
		return nil, &ErrProvider{Purpose: req.Purpose, Err: fmt.Errorf("llm: empty choices")}
	}

	p.logger.Info("llm_request_completed", append(logAttrs,
		"status", "success",
		"latency_ms", latency.Milliseconds(),
		"output_tokens", cr.Usage.CompletionTokens,
	)...)

	return &Response{
		Text:         cr.Choices[0].Message.Content,
		InputTokens:  cr.Usage.PromptTokens,
		OutputTokens: cr.Usage.CompletionTokens,
	}, nil
}

// formatLabel maps an internal response-format key to a human label for logs.
func formatLabel(f string) string {
	if f == "" {
		return "none"
	}
	return f
}

// providerErrorEnvelope is the OpenAI-compatible error body shape
// ({"error": {"code": ..., "message": ..., "type": ...}}) returned by DeepSeek
// and other providers on non-2xx responses.
type providerErrorEnvelope struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

// parseProviderError extracts the provider's machine-readable code and message
// from a non-2xx response body so failures are observable without dumping the
// full body. Unknown shapes yield empty strings (callers log the truncated
// body anyway).
func parseProviderError(body []byte) (code, message string) {
	var env providerErrorEnvelope
	if err := json.Unmarshal(body, &env); err != nil {
		return "", ""
	}
	return strings.TrimSpace(env.Error.Code), strings.TrimSpace(env.Error.Message)
}

// formatUnsupported reports whether the upstream rejected the requested
// response_format (a 4xx mentioning response_format/json_schema/format).
func formatUnsupported(err error) bool {
	pe, ok := err.(*ErrProvider)
	if !ok {
		return false
	}
	msg := strings.ToLower(pe.Err.Error())
	return strings.Contains(msg, "response_format") ||
		strings.Contains(msg, "json_schema") ||
		strings.Contains(msg, "response format") ||
		strings.Contains(msg, "json_object")
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// retryable reports whether an error should be retried by the runner.
// Transport failures and transient HTTP statuses (408/429/5xx) are retryable;
// definitively rejected requests (4xx client errors) are not. Proxy/config
// errors are surfaced at construction time, so they never reach this path.
func retryable(err error) bool {
	var pe *ErrProvider
	if ok := asErrProvider(err, &pe); ok {
		switch pe.StatusCode {
		case http.StatusRequestTimeout, http.StatusTooManyRequests,
			http.StatusInternalServerError, http.StatusBadGateway,
			http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		case http.StatusBadRequest, http.StatusUnauthorized, http.StatusForbidden,
			http.StatusNotFound, http.StatusUnprocessableEntity:
			return false
		default:
			// transport-level errors (StatusCode == 0) are transient
			return pe.StatusCode == 0
		}
	}
	return false
}

func asErrProvider(err error, target **ErrProvider) bool {
	if err == nil {
		return false
	}
	if pe, ok := err.(*ErrProvider); ok {
		*target = pe
		return true
	}
	return false
}

var _ = time.Second
