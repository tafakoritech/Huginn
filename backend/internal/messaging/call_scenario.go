package messaging

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"time"

	"github.com/tafakorpardazan/huginn/internal/llm"
)

// CallScenarioStrategy is the concise, structured sales reasoning that precedes
// the scenario. It records the LLM's lead↔project fit analysis conclusions —
// NOT hidden chain-of-thought, just the decision fields that drove the script.
type CallScenarioStrategy struct {
	LeadSummary    string   `json:"lead_summary"`
	Fit            string   `json:"fit"` // high | medium | low | uncertain
	FitSummary     string   `json:"fit_summary"`
	PrimaryAngle   string   `json:"primary_angle"`
	SecondaryAngle string   `json:"secondary_angle"`
	// LikelyPain is kept for backward compatibility. Prefer the explicit
	// KnownPains / PainHypotheses split below.
	LikelyPain string `json:"likely_pain"`
	// LikelyPainBasis classifies the likely_pain statement as "fact" (directly
	// supported by provided context) or "hypothesis" (plausible but unconfirmed).
	// "hypothesis" content must NEVER be spoken as a factual claim.
	LikelyPainBasis string `json:"likely_pain_basis,omitempty"`
	// ValidationRequired is true when LikelyPain is a hypothesis that must be
	// validated via discovery on the call rather than treated as known.
	ValidationRequired bool `json:"validation_required,omitempty"`
	// KnownPains are pain points directly supported by the provided context
	// (previous reports, notes, project pain points tied to evidence).
	KnownPains []string `json:"known_pains,omitempty"`
	// PainHypotheses are plausible pain points that must be converted into
	// discovery questions, never stated as facts to the prospect.
	PainHypotheses []string `json:"pain_hypotheses,omitempty"`
	Uncertainties  []string `json:"uncertainties"`
	CallObjective  string   `json:"call_objective"`
	Tone           string   `json:"tone"`
}

// DiscoveryQuestion pairs a question with the strategic purpose it serves so
// the salesperson knows why each question is being asked for THIS lead.
type DiscoveryQuestion struct {
	Question string `json:"question"`
	Purpose  string `json:"purpose"`
}

// ConversationBranch is a contingency strategy for a likely call outcome.
type ConversationBranch struct {
	Condition        string `json:"condition"`
	ResponseStrategy string `json:"response_strategy"`
}

// Generation status values exposed on every scenario so clients can never
// mistake deterministic fallback content for a personalized LLM generation.
const (
	// GenerationStatusGenerated means the scenario was produced by the LLM and
	// passed structural validation.
	GenerationStatusGenerated = "generated"
	// GenerationStatusDegraded means the LLM failed or its output was rejected,
	// so the returned scenario is the deterministic fallback. Clients must
	// surface this to the agent instead of presenting it as AI-generated.
	GenerationStatusDegraded = "degraded"
)

// CallScenario is a practical, spoken phone-call script generated from the
// lead's real context and its project's sales context. An agent can read it
// right before or during the call.
type CallScenario struct {
	ID                   string                `json:"id,omitempty"`
	GenerationID         string                `json:"generation_id,omitempty"`
	LeadID               string                `json:"lead_id,omitempty"`
	ContactID            *string               `json:"contact_id,omitempty"`
	UserID               *string               `json:"user_id,omitempty"`
	ProjectID            *string               `json:"project_id,omitempty"`
	SalesAngle           string                `json:"sales_angle,omitempty"`
	OpeningSentence      string                `json:"opening_sentence"`
	Introduction         string                `json:"introduction"`
	ReasonForCall        string                `json:"reason_for_call"`
	ValueProposition     string                `json:"value_proposition"`
	DiscoveryQuestions   []string              `json:"discovery_questions"`
	LikelyObjections     []string              `json:"likely_objections"`
	ObjectionResponses   []string              `json:"objection_responses"`
	RecommendedNextStep  string                `json:"recommended_next_step"`
	ClosingLine          string                `json:"closing_line"`
	ContextUsed          ContextUsed           `json:"context_used"`
	Strategy             *CallScenarioStrategy `json:"strategy,omitempty"`
	DiscoveryPurposes    []DiscoveryQuestion   `json:"discovery_question_purposes,omitempty"`
	ConversationBranches []ConversationBranch  `json:"conversation_branches,omitempty"`
	IsGenerated          bool                  `json:"is_generated,omitempty"`
	// GenerationStatus is "generated" or "degraded" (see constants). It makes
	// fallback usage explicit to API clients.
	GenerationStatus string `json:"generation_status,omitempty"`
	// FallbackUsed is true when the scenario is the deterministic template
	// (because the LLM failed or its output was rejected). Always serialized:
	// clients must see an explicit false rather than an absent field.
	FallbackUsed bool `json:"fallback_used"`
	// GenerationError carries a compact human-readable reason when the LLM
	// could not produce the scenario (empty when generation succeeded).
	GenerationError string    `json:"generation_error,omitempty"`
	CreatedAt       time.Time `json:"created_at,omitempty"`
	UpdatedAt       time.Time `json:"updated_at,omitempty"`
}

// ContextUsed is the LLM's own statement of which project and lead context it
// actually used to build the scenario. It is part of the required output schema
// so every generation can be debugged against the context that was sent.
type ContextUsed struct {
	Project    string `json:"project"`
	Lead       string `json:"lead"`
	SalesAngle string `json:"sales_angle"`
}

// GenerationOptions makes every generation request unique and scoped to the
// exact lead + project. The same options must never be reused: each request
// gets a fresh GenerationID and Timestamp so no two prompts are identical.
type GenerationOptions struct {
	GenerationID string
	LeadID       string
	ProjectID    string
	Timestamp    time.Time
	// SalesAngle is an optional client-provided strategic focus hint. When
	// empty, the model selects the strongest angle for THIS lead itself.
	SalesAngle string
	// PreviousAngle is the primary_angle of the most recent generation for the
	// same lead. It is passed only for "Generate Again" so the model can
	// consciously propose a different or sharper strategy — reasoning-based
	// variation instead of template rotation.
	PreviousAngle string
}

// ScenarioGenerator produces a call scenario for a lead.
type ScenarioGenerator interface {
	GenerateScenario(ctx context.Context, leadCtx *LeadContext, opts GenerationOptions) (*CallScenario, error)
}

// CallScenarioGenerator generates call scenarios via the LLM.
type CallScenarioGenerator struct {
	runner  llm.Runner
	sender  *SenderProfile
	prompts PromptSettingsProvider
	logger  *slog.Logger
	// opTimeout bounds the ENTIRE generation operation (LLM attempt + retries +
	// repair attempts + parsing) so the handler always writes its response
	// before the HTTP server WriteTimeout. It must be longer than the LLM
	// per-attempt timeout but shorter than WriteTimeout.
	opTimeout time.Duration
	// lastSemanticFlags holds non-blocking quality observations from the most
	// recent generation so the completion log can surface them.
	lastSemanticFlags []string
	// lastLLMDurationMs and lastInputChars capture the most recent generation's
	// upstream latency and prompt size so the completion log can correlate the
	// full operation with the underlying LLM call.
	lastLLMDurationMs int64
	lastInputChars    int
}

// defaultScenarioOperationTimeout bounds the whole generation when no explicit
// timeout is configured. It is slightly longer than the default LLM per-attempt
// timeout (LLM_TIMEOUT_SECONDS=180) and well below the server WriteTimeout.
const defaultScenarioOperationTimeout = 185 * time.Second

// GeneratorOption customizes a CallScenarioGenerator at construction.
type GeneratorOption func(*CallScenarioGenerator)

// WithLogger attaches a structured logger for scenario generation diagnostics.
func WithLogger(l *slog.Logger) GeneratorOption {
	return func(g *CallScenarioGenerator) {
		if l != nil {
			g.logger = l
		}
	}
}

// WithOperationTimeout bounds the total wall-clock budget for one call-scenario
// generation (LLM attempts + retries + repairs + parsing). The value must sit
// between the LLM per-attempt timeout and the HTTP server WriteTimeout.
func WithOperationTimeout(d time.Duration) GeneratorOption {
	return func(g *CallScenarioGenerator) {
		if d > 0 {
			g.opTimeout = d
		}
	}
}

func NewCallScenarioGenerator(runner llm.Runner, sender *SenderProfile, prompts PromptSettingsProvider, opts ...GeneratorOption) *CallScenarioGenerator {
	g := &CallScenarioGenerator{runner: runner, sender: sender, prompts: prompts, logger: slog.Default(), opTimeout: defaultScenarioOperationTimeout}
	for _, o := range opts {
		o(g)
	}
	return g
}

// scenarioPromptVersion identifies the prompt definition driving generation.
// Bump it whenever the prompt, context serialization, or output schema changes.
const scenarioPromptVersion = "huginn-call-scenario-v4"

// SalesAngleFocuses is the vocabulary of strategic focus hints a client may
// pass via the sales_angle request parameter. It is NOT rotated automatically:
// the model chooses the primary angle from the lead analysis. A focus is a
// client preference that the model may override with a stronger lead-specific
// argument.
var SalesAngleFocuses = []string{
	"Focus the conversation on increasing the lead's customer engagement and growing its business.",
	"Focus the conversation on reducing the lead's operational or business cost.",
	"Focus the conversation on creating a durable competitive advantage over the lead's competitors.",
	"Focus the conversation on the most probable business pain of THIS exact lead and how the project's value proposition relieves it.",
	"Focus the conversation on a quick, low-risk pilot that proves the project's value with minimal commitment.",
}

func (g *CallScenarioGenerator) GenerateScenario(ctx context.Context, leadCtx *LeadContext, opts GenerationOptions) (*CallScenario, error) {
	start := time.Now()
	// Project context is mandatory. Without it the model would fall back to a
	// generic script, so generation FAILS instead of producing one.
	if leadCtx == nil || leadCtx.Project == nil || strings.TrimSpace(leadCtx.Project.Title) == "" {
		return nil, ErrNoProject
	}

	g.logContext(leadCtx, opts, "scenario_generation_started", false)

	system := DefaultCallScenarioSystemPrompt
	promptSource := "builtin"
	if g.prompts != nil {
		if p, err := g.prompts.Load(ctx); err == nil && p != nil {
			if p.CallScenarioSystemPrompt != "" {
				system = p.CallScenarioSystemPrompt
				promptSource = "stored"
			}
			if p.Tone != "" {
				system = withTone(system, p.Tone)
			}
		}
	}
	// Contract drift guard: the stored settings prompt is editable at runtime,
	// but it MUST describe the canonical v3 output contract. A legacy prompt
	// (e.g. the old 9-field shape without strategy/context_used) silently
	// changes what the model emits; flag it so operators notice instead of
	// getting opaque rejections downstream.
	if !strings.Contains(system, "context_used") || !strings.Contains(system, "strategy") {
		g.logger.Warn("scenario_prompt_contract_drift",
			"prompt_source", promptSource,
			"lead_id", leadCtx.LeadID,
			"generation_id", opts.GenerationID,
			"prompt_version", scenarioPromptVersion,
			"expected_contract", "v4 (strategy + known_pains/pain_hypotheses + context_used + conversation fields)",
			"hint", "stored call_scenario_system_prompt does not match the v4 output schema; update it via the settings API or remove it to use the built-in prompt",
		)
	}
	// Bound the ENTIRE generation (LLM attempt + retries + repair attempts +
	// parsing) so the handler always completes before the HTTP server
	// WriteTimeout. The operation context drops any enclosing wall-clock
	// deadline (a child context with 180s does not help if its parent is
	// cancelled after 30s) but keeps cancellation: a client that leaves the
	// screen (or loses network) still cancels the generation intentionally. This
	// is the deliberate inner/outer timeout hierarchy.
	opCtx, cancel := context.WithTimeout(llm.WithoutDeadline(ctx), g.opTimeout)
	defer cancel()
	out, err := g.generate(opCtx, leadCtx, system, opts)
	if err != nil {
		// The LLM was unavailable, refused, or returned output that failed
		// structural validation. Return the deterministic project-aware
		// scenario so the agent is never blocked, but make the degradation
		// EXPLICIT: generation_status, fallback_used and the rejection reasons
		// are surfaced on the scenario and in the logs so template-like output
		// can never be mistaken for a personalized LLM generation.
		reasons := rejectionReasons(err)
		g.logger.Error("call_scenario_generation_failed",
			"project_id", projectID(leadCtx),
			"lead_id", leadCtx.LeadID,
			"generation_id", opts.GenerationID,
			"prompt_version", scenarioPromptVersion,
			"error", errString(err),
			"reasons", reasons,
			"llm_duration_ms", g.lastLLMDurationMs,
			"input_chars", g.lastInputChars,
			"fallback_used", true,
			"generation_status", GenerationStatusDegraded,
			"total_latency_ms", time.Since(start).Milliseconds(),
		)
		out = fallbackScenario(leadCtx, g.sender)
		applyGenerationMeta(out, leadCtx, opts)
		out.IsGenerated = false
		out.GenerationStatus = GenerationStatusDegraded
		out.FallbackUsed = true
		out.GenerationError = compactRejection(err, reasons)
		return out, nil
	}
	applyGenerationMeta(out, leadCtx, opts)
	out.IsGenerated = true
	out.GenerationStatus = GenerationStatusGenerated
	out.FallbackUsed = false
	known, unknown := leadCtx.coverage()
	g.logger.Info("call_scenario_generation_completed",
		"project_id", projectID(leadCtx),
		"lead_id", leadCtx.LeadID,
		"generation_id", opts.GenerationID,
		"prompt_version", scenarioPromptVersion,
		"input_chars", g.lastInputChars,
		"known_field_count", len(known),
		"unknown_field_count", len(unknown),
		"reports_count", reportCount(leadCtx),
		"interactions_count", interactionCount(leadCtx),
		"llm_duration_ms", g.lastLLMDurationMs,
		"parse_result", "success",
		"validation_result", "usable",
		"validation_warnings", g.lastSemanticFlags,
		"fallback_used", false,
		"generation_status", GenerationStatusGenerated,
		"total_latency_ms", time.Since(start).Milliseconds(),
		"strategy_present", out.Strategy != nil,
		"discovery_count", len(out.DiscoveryQuestions),
		"objection_count", len(out.LikelyObjections),
		"primary_angle", strategyAngle(out),
		"semantic_flags", g.lastSemanticFlags,
	)
	return out, nil
}

func (g *CallScenarioGenerator) generate(ctx context.Context, leadCtx *LeadContext, system string, opts GenerationOptions) (*CallScenario, error) {
	userPrompt := buildScenarioPrompt(g.sender, leadCtx, opts)
	req := llm.Request{
		Purpose:       llm.PurposeCallScenario,
		PromptVersion: scenarioPromptVersion,
		EntityType:    "lead",
		EntityID:      leadCtx.LeadID,
		SystemPrompt:  system,
		UserPrompt:    userPrompt,
		JSONSchema:    scenarioJSONSchema(),
		Temperature:   scenarioTemperature(),
		Meta: map[string]string{
			"lead_id":       leadCtx.LeadID,
			"project_id":    leadCtx.Project.ProjectID,
			"generation_id": opts.GenerationID,
		},
	}
	g.lastInputChars = len(userPrompt) + len(system)

	var out CallScenario
	run, err := g.runner.GenerateStructured(ctx, req, &out)
	if run != nil {
		g.lastLLMDurationMs = run.LatencyMs
	}
	if err != nil {
		return nil, err
	}
	if run == nil || run.Status != "success" {
		return nil, &ErrScenarioRejected{Reasons: []string{"provider_run_" + runStatus(run)}}
	}

	// Normalize cosmetic gaps the model is allowed to omit (context_used,
	// strategy) BEFORE structural validation, then validate. Structural
	// invalidity rejects; semantic quality is logged and surfaced, not silently
	// discarded.
	g.lastSemanticFlags = nil
	validation := normalizeScenario(&out, leadCtx, opts)
	if len(validation.Semantic) > 0 {
		g.lastSemanticFlags = validation.Semantic
		g.logger.Warn("scenario_output_semantic_flags",
			"lead_id", leadCtx.LeadID,
			"generation_id", opts.GenerationID,
			"semantic", validation.Semantic,
		)
	}
	if !validation.Usable() {
		return nil, &ErrScenarioRejected{Reasons: validation.Structural}
	}
	return &out, nil
}

// ErrScenarioRejected is returned when the provider succeeded but the parsed
// output failed structural validation. Reasons are the actionable diagnostics
// (e.g. "opening_sentence_empty", "objection_responses_not_aligned") that
// replace the opaque "unusable_or_empty_output".
type ErrScenarioRejected struct {
	Reasons []string
}

func (e *ErrScenarioRejected) Error() string {
	return "scenario_output_rejected: " + strings.Join(e.Reasons, "; ")
}

func rejectionReasons(err error) []string {
	var rej *ErrScenarioRejected
	if errors.As(err, &rej) {
		return rej.Reasons
	}
	return []string{errString(err)}
}

func compactRejection(err error, reasons []string) string {
	if len(reasons) == 1 && reasons[0] != errString(err) {
		return reasons[0]
	}
	if len(reasons) > 1 {
		return strings.Join(reasons, "; ")
	}
	return errString(err)
}

func runStatus(run *llm.Run) string {
	if run == nil {
		return "nil"
	}
	return run.Status
}

func scenarioTemperature() *float64 {
	t := 0.7
	return &t
}

func projectID(c *LeadContext) string {
	if c == nil || c.Project == nil {
		return ""
	}
	return c.Project.ProjectID
}

func strategyAngle(s *CallScenario) string {
	if s == nil || s.Strategy == nil {
		return ""
	}
	return s.Strategy.PrimaryAngle
}

func errString(err error) string {
	if err == nil {
		return "no scenario produced"
	}
	return err.Error()
}

func (g *CallScenarioGenerator) logContext(leadCtx *LeadContext, opts GenerationOptions, msg string, fallback bool) {
	known, unknown := leadCtx.coverage()
	g.logger.Info(msg,
		"project_id", projectID(leadCtx),
		"lead_id", leadCtx.LeadID,
		"generation_id", opts.GenerationID,
		"prompt_version", scenarioPromptVersion,
		"known_fields", strings.Join(known, "|"),
		"unknown_fields", strings.Join(unknown, "|"),
		"interactions_count", interactionCount(leadCtx),
		"reports_count", reportCount(leadCtx),
		"previous_angle", opts.PreviousAngle,
		"sales_angle_hint", opts.SalesAngle,
		"fallback_used", fallback,
	)
}

func interactionCount(c *LeadContext) int {
	if c == nil || c.PastInteractions == "" {
		return 0
	}
	return strings.Count(c.PastInteractions, "\n") + 1
}

func reportCount(c *LeadContext) int {
	if c == nil || c.PreviousReports == "" {
		return 0
	}
	return strings.Count(c.PreviousReports, "\n") + 1
}

// scenarioJSONSchema is the strict structured-output schema for the scenario
// call. The strategy block comes first so the model reasons before writing.
func scenarioJSONSchema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"strategy": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"lead_summary":        map[string]any{"type": "string"},
					"fit":                 map[string]any{"type": "string", "enum": []string{"high", "medium", "low", "uncertain"}},
					"fit_summary":         map[string]any{"type": "string"},
					"primary_angle":       map[string]any{"type": "string"},
					"secondary_angle":     map[string]any{"type": "string"},
					"likely_pain":         map[string]any{"type": "string"},
					"likely_pain_basis":   map[string]any{"type": "string", "enum": []string{"fact", "hypothesis"}},
					"validation_required": map[string]any{"type": "boolean"},
					"known_pains":         map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"pain_hypotheses":     map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"uncertainties":       map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
					"call_objective":      map[string]any{"type": "string"},
					"tone":                map[string]any{"type": "string"},
				},
				"required": []string{"lead_summary", "fit", "fit_summary", "primary_angle", "secondary_angle", "likely_pain", "uncertainties", "call_objective", "tone"},
			},
			"opening_sentence":    map[string]any{"type": "string"},
			"introduction":        map[string]any{"type": "string"},
			"reason_for_call":     map[string]any{"type": "string"},
			"value_proposition":   map[string]any{"type": "string"},
			"discovery_questions": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"discovery_question_purposes": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"question": map[string]any{"type": "string"},
						"purpose":  map[string]any{"type": "string"},
					},
					"required": []string{"question", "purpose"},
				},
			},
			"likely_objections":   map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"objection_responses": map[string]any{"type": "array", "items": map[string]any{"type": "string"}},
			"conversation_branches": map[string]any{
				"type": "array",
				"items": map[string]any{
					"type": "object",
					"properties": map[string]any{
						"condition":         map[string]any{"type": "string"},
						"response_strategy": map[string]any{"type": "string"},
					},
					"required": []string{"condition", "response_strategy"},
				},
			},
			"recommended_next_step": map[string]any{"type": "string"},
			"closing_line":          map[string]any{"type": "string"},
			"context_used": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"project":     map[string]any{"type": "string"},
					"lead":        map[string]any{"type": "string"},
					"sales_angle": map[string]any{"type": "string"},
				},
				"required": []string{"project", "lead", "sales_angle"},
			},
		},
		"required": []string{
			"strategy", "opening_sentence", "introduction", "reason_for_call", "value_proposition",
			"discovery_questions", "discovery_question_purposes", "likely_objections", "objection_responses",
			"conversation_branches", "recommended_next_step", "closing_line", "context_used",
		},
	}
}

// buildScenarioPrompt renders the complete user prompt for a scenario
// generation: a unique request envelope, an optional variety hint, the SENDER
// block, and the RECIPIENT block that always carries the PROJECT CONTEXT and
// LEAD INFORMATION.
func buildScenarioPrompt(sender *SenderProfile, leadCtx *LeadContext, opts GenerationOptions) string {
	var b strings.Builder
	b.WriteString("GENERATION REQUEST (this is a UNIQUE request; NEVER reuse, repeat, or reproduce any previous output, cached response, or template):\n")
	b.WriteString("generation_id: " + opts.GenerationID + "\n")
	b.WriteString("timestamp: " + opts.Timestamp.UTC().Format(time.RFC3339) + "\n")
	if leadCtx != nil {
		b.WriteString("lead_id: " + leadCtx.LeadID + "\n")
		if leadCtx.Project != nil {
			b.WriteString("project_id: " + leadCtx.Project.ProjectID + "\n")
		}
	}
	if strings.TrimSpace(opts.SalesAngle) != "" {
		b.WriteString("sales_focus (client preference; you may override it only with a stronger lead-specific argument): " + opts.SalesAngle + "\n")
	} else {
		b.WriteString("sales_focus: none — choose the strongest angle for THIS lead yourself.\n")
	}
	if strings.TrimSpace(opts.PreviousAngle) != "" {
		b.WriteString("\nPREVIOUS GENERATION (this is a \"Generate Again\" request):\n")
		b.WriteString("The previously chosen primary angle was: " + opts.PreviousAngle + "\n")
		b.WriteString("Re-analyze this prospect. If a different angle is genuinely stronger for this specific lead, choose it. Otherwise sharpen the same angle. NEVER merely reword the previous output.\n")
		b.WriteString("--- END PREVIOUS GENERATION ---\n")
	}

	b.WriteString("\nSENDER (our company — the caller):\n")
	if sender == nil || sender.IsZero() {
		b.WriteString("Company: [unset]\n")
		b.WriteString("(The sender's company is not disclosed. Introduce yourself simply as a sales representative; do NOT name any company and do NOT claim the sender works in the lead's industry.)\n")
	} else {
		if sender.CompanyName != "" {
			b.WriteString("Company: " + sender.CompanyName + "\n")
		}
		if sender.Description != "" {
			b.WriteString("Company description / what we offer: " + truncateStr(sender.Description, 500) + "\n")
		}
		if s := sender.representative(); s != "" {
			b.WriteString("Representative: " + s + "\n")
		}
	}
	b.WriteString("\nRECIPIENT (the lead — the person/business being called):\n")
	if leadCtx != nil {
		if rendered := leadCtx.Render(); rendered != "" {
			b.WriteString(rendered + "\n")
		}
	}
	b.WriteString("\nAnalyze this specific prospect against this specific project, choose a sales strategy, and generate a call scenario that could realistically happen with THIS prospect. Use ONLY the context above; never invent facts.")
	return b.String()
}

// applyGenerationMeta stamps the per-generation metadata onto a scenario and
// guarantees the context_used debug block is always populated with real data
// even when the model omits it. The effective sales angle is the client focus
// when provided, otherwise the model-selected primary_angle.
func applyGenerationMeta(s *CallScenario, leadCtx *LeadContext, opts GenerationOptions) {
	if s == nil {
		return
	}
	if leadCtx != nil {
		s.LeadID = leadCtx.LeadID
	}
	s.GenerationID = opts.GenerationID
	angle := strings.TrimSpace(opts.SalesAngle)
	if angle == "" {
		angle = strategyAngle(s)
	}
	s.SalesAngle = angle
	if strings.TrimSpace(s.ContextUsed.Project) == "" {
		if leadCtx != nil && leadCtx.Project != nil {
			s.ContextUsed.Project = leadCtx.Project.Title
		}
	}
	if strings.TrimSpace(s.ContextUsed.Lead) == "" {
		if leadCtx != nil {
			s.ContextUsed.Lead = leadCtx.CompanyName
		}
	}
	if strings.TrimSpace(s.ContextUsed.SalesAngle) == "" {
		s.ContextUsed.SalesAngle = angle
	}
}

// ScenarioValidation separates structural validity from semantic quality.
//
// Structural problems make the scenario unusable and REJECT the generation
// (the deterministic fallback is returned and degradation is flagged).
//
// Semantic problems are observable quality flags: they are logged and surfaced
// on the scenario, but they do NOT by themselves discard otherwise-usable model
// output — cosmetic differences must never throw away a good generation.
type ScenarioValidation struct {
	Structural []string `json:"structural,omitempty"`
	Semantic   []string `json:"semantic,omitempty"`
}

// Usable reports whether the scenario passes structural validation.
func (v ScenarioValidation) Usable() bool { return len(v.Structural) == 0 }

// Reasons are the hard failures that make the scenario unusable.
func (v ScenarioValidation) Reasons() []string { return v.Structural }

// Warnings are the soft quality flags that do not block the generation.
func (v ScenarioValidation) Warnings() []string { return v.Semantic }

// normalizeScenario repairs the cosmetic gaps the model is allowed to omit and
// then validates the scenario:
//
//   - context_used is always filled from real project/lead data when the model
//     omitted it (the model WAS given that context, so recording it is
//     accurate), and the omission is recorded as a semantic flag.
//   - strategy is synthesized from the generated content when absent
//     (ensureStrategy), so a missing optional block never discards good output.
func normalizeScenario(s *CallScenario, leadCtx *LeadContext, opts GenerationOptions) ScenarioValidation {
	if s == nil {
		return ScenarioValidation{Structural: []string{"scenario_nil"}}
	}

	// 1. Repair context_used before validation: it is a debug block that
	//    applyGenerationMeta would fill anyway, so a missing/partial block must
	//    not reject an otherwise complete scenario.
	ctxOmitted := strings.TrimSpace(s.ContextUsed.Project) == "" && strings.TrimSpace(s.ContextUsed.Lead) == ""
	angle := strings.TrimSpace(opts.SalesAngle)
	if angle == "" {
		angle = strategyAngle(s)
	}
	if strings.TrimSpace(s.ContextUsed.Project) == "" {
		if leadCtx != nil && leadCtx.Project != nil {
			s.ContextUsed.Project = leadCtx.Project.Title
		}
	}
	if strings.TrimSpace(s.ContextUsed.Lead) == "" {
		if leadCtx != nil {
			s.ContextUsed.Lead = leadCtx.CompanyName
		}
	}
	if strings.TrimSpace(s.ContextUsed.SalesAngle) == "" {
		s.ContextUsed.SalesAngle = angle
	}

	// 2. Repair a missing strategy block (optional) by synthesizing it from the
	//    generated content. Never discard good LLM output over a missing
	//    optional field.
	ensureStrategy(s, leadCtx, opts)
	normalizeStrategyEvidence(s)

	v := validateScenario(s)
	if ctxOmitted {
		v.Semantic = append(v.Semantic, "context_used_omitted_by_model")
	}
	return v
}

// normalizeStrategyEvidence fills safe defaults for the evidence-discipline
// fields of the strategy so downstream consumers never have to guess. The safe
// default for an unclassified likely pain is "hypothesis" (never assume a fact
// the model did not label).
func normalizeStrategyEvidence(s *CallScenario) {
	if s.Strategy == nil {
		return
	}
	if strings.TrimSpace(s.Strategy.LikelyPainBasis) == "" && strings.TrimSpace(s.Strategy.LikelyPain) != "" {
		s.Strategy.LikelyPainBasis = "hypothesis"
	}
	if s.Strategy.LikelyPainBasis == "hypothesis" && strings.TrimSpace(s.Strategy.LikelyPain) != "" {
		s.Strategy.ValidationRequired = true
	}
	if len(s.Strategy.KnownPains) == 0 && len(s.Strategy.PainHypotheses) == 0 && strings.TrimSpace(s.Strategy.LikelyPain) != "" && s.Strategy.LikelyPainBasis != "fact" {
		// A legacy/omitted split: treat the single likely_pain as a hypothesis
		// so it is never silently consumed as a confirmed fact.
		s.Strategy.PainHypotheses = []string{s.Strategy.LikelyPain}
	}
}

// validateScenario checks structural validity and semantic quality. Structural
// reasons are returned in a stable, actionable order so diagnostics like
// "opening_sentence_empty" replace the old opaque boolean rejection.
func validateScenario(s *CallScenario) ScenarioValidation {
	var v ScenarioValidation
	if s == nil {
		return ScenarioValidation{Structural: []string{"scenario_nil"}}
	}
	requireText := func(field, value string) {
		if strings.TrimSpace(value) == "" {
			v.Structural = append(v.Structural, field+"_empty")
		}
	}

	// Required conversation content.
	requireText("opening_sentence", s.OpeningSentence)
	requireText("introduction", s.Introduction)
	requireText("reason_for_call", s.ReasonForCall)
	requireText("value_proposition", s.ValueProposition)
	requireText("recommended_next_step", s.RecommendedNextStep)
	requireText("closing_line", s.ClosingLine)

	// Required arrays.
	if len(s.DiscoveryQuestions) == 0 {
		v.Structural = append(v.Structural, "discovery_questions_empty")
	}
	if len(s.LikelyObjections) == 0 {
		v.Structural = append(v.Structural, "likely_objections_empty")
	}
	if len(s.ObjectionResponses) == 0 {
		v.Structural = append(v.Structural, "objection_responses_empty")
	}
	if len(s.DiscoveryQuestions) != 0 && len(s.DiscoveryPurposes) != 0 && len(s.DiscoveryPurposes) != len(s.DiscoveryQuestions) {
		v.Semantic = append(v.Semantic, "discovery_purposes_not_aligned_with_questions")
	}

	// Objections and responses must be coherent and aligned.
	if len(s.LikelyObjections) != len(s.ObjectionResponses) {
		v.Structural = append(v.Structural, "objection_responses_not_aligned")
	}
	for _, resp := range s.ObjectionResponses {
		if strings.TrimSpace(resp) == "" {
			v.Structural = append(v.Structural, "objection_response_empty")
			break
		}
	}
	for _, q := range s.DiscoveryQuestions {
		if strings.TrimSpace(q) == "" {
			v.Structural = append(v.Structural, "discovery_question_empty")
			break
		}
	}

	// Strategy block is repaired by ensureStrategy, but the synthesized result
	// must still be usable: primary angle and call objective are the strategic
	// core that downstream agents rely on.
	if s.Strategy == nil {
		v.Structural = append(v.Structural, "strategy_missing")
	} else {
		if strings.TrimSpace(s.Strategy.PrimaryAngle) == "" {
			v.Structural = append(v.Structural, "strategy.primary_angle_empty")
		}
		if strings.TrimSpace(s.Strategy.CallObjective) == "" {
			v.Structural = append(v.Structural, "strategy.call_objective_empty")
		}
	}

	// ---- Semantic (observable, non-blocking) checks ----
	if len(s.OpeningSentence) > 0 && len([]rune(s.OpeningSentence)) < 8 {
		v.Semantic = append(v.Semantic, "opening_too_short")
	}
	if len(s.DiscoveryQuestions) > 0 && len(s.DiscoveryQuestions) < 2 {
		v.Semantic = append(v.Semantic, "discovery_questions_too_few")
	}
	if placeholderIn(s.OpeningSentence, s.Introduction, s.ValueProposition, s.ClosingLine) {
		v.Semantic = append(v.Semantic, "template_placeholder_left_in_output")
	}

	// Strategy ↔ scenario consistency: the strategy, opening, discovery and
	// next step must tell the same story. These are SOFT warnings: an uncertain
	// fit that drifts into a pitch, or a qualification objective that jumps to a
	// contract, is flagged for review but never discards a usable scenario.
	v.appendConsistencyFlags(s)
	return v
}

// appendConsistencyFlags adds soft warnings when the strategy and the scenario
// body tell different stories (e.g. uncertain fit but an aggressive pitch, or a
// qualification objective that recommends a contract). These are warnings, not
// hard failures: the caller decides whether to reuse the output.
func (v *ScenarioValidation) appendConsistencyFlags(s *CallScenario) {
	if s.Strategy == nil {
		return
	}
	fit := strings.ToLower(strings.TrimSpace(s.Strategy.Fit))
	objective := strings.ToLower(strings.TrimSpace(s.Strategy.CallObjective))
	next := strings.ToLower(strings.TrimSpace(s.RecommendedNextStep))
	value := strings.ToLower(strings.TrimSpace(s.ValueProposition))

	// A fit of "uncertain" must not be combined with a hard pitch or a
	// guaranteed outcome in the value proposition.
	if fit == "uncertain" {
		if pitchLike(value) {
			v.Semantic = append(v.Semantic, "uncertain_fit_but_pitching_value")
		}
		if nextContractLike(next) {
			v.Semantic = append(v.Semantic, "uncertain_fit_but_contract_next_step")
		}
	}

	// A qualification objective must not jump straight to a contract/proposal.
	if qualifyLike(objective) && nextContractLike(next) {
		v.Semantic = append(v.Semantic, "qualification_objective_but_contract_next_step")
	}

	// A discovery-first objective that contains zero open-ended discovery
	// questions suggests the scenario is not actually discovery-oriented.
	if discoverLike(objective) && len(s.DiscoveryQuestions) == 0 {
		v.Semantic = append(v.Semantic, "discovery_objective_without_discovery_questions")
	}

	// Unsupported pain hypotheses: when the strategy marks the likely pain as a
	// hypothesis (or lists it in pain_hypotheses) the customer-facing text must
	// not state it as a confirmed fact.
	if (s.Strategy.LikelyPainBasis == "hypothesis" || len(s.Strategy.PainHypotheses) > 0) && strings.TrimSpace(s.Strategy.LikelyPain) != "" {
		v.Semantic = append(v.Semantic, "unsupported_pain_hypothesis_stated_as_fact")
	}
}

// pitchLike detects an aggressive "we will make you X" guaranteed-outcome claim.
func pitchLike(value string) bool {
	markers := []string{"۳ برابر", "3 برابر", "۳x", "3x", "قطعاً", "تضمینی", "صددرصد", "۱۰۰٪", "guaranteed", "3x", "دوبرابر", "۲ برابر"}
	for _, m := range markers {
		if strings.Contains(value, m) {
			return true
		}
	}
	return false
}

func nextContractLike(next string) bool {
	markers := []string{"قرارداد", "contract", "پروپوزال", "proposal", "امضای", "درخواست خرید", "توافقنامه"}
	for _, m := range markers {
		if strings.Contains(next, m) {
			return true
		}
	}
	return false
}

func qualifyLike(objective string) bool {
	markers := []string{"qualif", "کیفیکاسیون", "شناسایی", "کشف", "discovery", "qualification", "اعتبارسنجی"}
	for _, m := range markers {
		if strings.Contains(objective, m) {
			return true
		}
	}
	return false
}

func discoverLike(objective string) bool {
	return qualifyLike(objective)
}

// placeholderIn detects obviously un-filled template markers (square-bracket
// placeholders, generic "company name" stubs) that indicate broken template
// output rather than a real generation.
func placeholderIn(fields ...string) bool {
	markers := []string{"[اسم", "[نام", "[Company", "[company", "[YOUR", "نام شرکت", "اسم شما", "X X X", "<<", ">>", "???", "[INSERT"}
	for _, f := range fields {
		if f == "" {
			continue
		}
		lower := strings.ToLower(f)
		for _, m := range markers {
			if strings.Contains(lower, strings.ToLower(m)) {
				return true
			}
		}
	}
	return false
}

// ensureStrategy guarantees every returned scenario carries a strategy block.
// When the model omitted it, a minimal strategy is synthesized from the
// generated content so downstream consumers always see the decision fields.
// It returns true when the strategy came from the model.
func ensureStrategy(s *CallScenario, leadCtx *LeadContext, opts GenerationOptions) bool {
	if s.Strategy != nil && strings.TrimSpace(s.Strategy.PrimaryAngle) != "" {
		return true
	}
	st := &CallScenarioStrategy{
		Fit:           "uncertain",
		Uncertainties: []string{},
		Tone:          "natural, concise, spoken language",
	}
	if leadCtx != nil {
		st.LeadSummary = strings.TrimSpace(joinNonEmpty(leadCtx.CompanyName, leadCtx.Industry, leadCtx.City))
	}
	st.FitSummary = "fit was not separately scored by the model; treat the value proposition as the working hypothesis and validate it with discovery questions."
	st.PrimaryAngle = strings.TrimSpace(s.ValueProposition)
	if st.PrimaryAngle == "" {
		st.PrimaryAngle = strings.TrimSpace(s.ReasonForCall)
	}
	st.SecondaryAngle = ""
	st.LikelyPain = "to be confirmed through discovery on the call (no evidence was provided for a specific pain)."
	st.LikelyPainBasis = "hypothesis"
	st.ValidationRequired = true
	st.PainHypotheses = []string{}
	st.KnownPains = []string{}
	st.CallObjective = strings.TrimSpace(s.RecommendedNextStep)
	if strings.TrimSpace(st.LeadSummary) == "" {
		st.LeadSummary = "lead"
	}
	s.Strategy = st
	return false
}

// fallbackScenario is a deterministic, zero-fabrication scenario used when the
// LLM is unavailable. It only references real lead and project data, so it is
// never a generic template: the project's title, description, and value
// proposition are woven into the script when present.
func fallbackScenario(leadCtx *LeadContext, sender *SenderProfile) *CallScenario {
	company := ""
	display := ""
	var proj *ProjectContext
	if leadCtx != nil {
		company = leadCtx.CompanyName
		display = leadCtx.DisplayName
		proj = leadCtx.Project
	}
	target := company
	if target == "" {
		target = display
	}
	if target == "" {
		target = "شما"
	}
	senderCompany := ""
	if sender != nil {
		senderCompany = sender.CompanyName
	}

	projTitle, projDesc, projValue, projPain, projObjections := "", "", "", "", ""
	if proj != nil {
		projTitle = strings.TrimSpace(proj.Title)
		projDesc = strings.TrimSpace(proj.Description)
		projValue = strings.TrimSpace(proj.ValueProposition)
		projPain = strings.TrimSpace(proj.PainPoints)
		projObjections = strings.TrimSpace(proj.CommonObjections)
	}

	opening := "سلام، وقت شما بخیر. مزاحم می‌شوم."
	if target != "شما" {
		opening = "سلام، وقت شما بخیر. با " + target + " در تماسم."
	}
	intro := "اینجا " + senderCompany + " است؛ مجموعه ما در زمینه همکاری تجاری با تولیدکنندگان و توزیع‌کنندگان فعال است."
	if projTitle != "" {
		intro = "اینجا " + senderCompany + " است؛ در قالب پروژه «" + projTitle + "» با شما تماس می‌گیریم."
	}
	reason := "دلیل تماسم این است که مجموعه شما را برای بررسی امکان همکاری شناسایی کرده‌ایم و می‌خواستیم بیشتر آشنا شویم."
	if projDesc != "" {
		reason = "دلیل تماسم این است که در چارچوب «" + truncateStr(projDesc, 200) + "» فکر می‌کردیم همکاری با مجموعه شما می‌تواند برای هر دو طرف ارزش‌آفرین باشد."
	}
	value := "اگر فرصتی باشد، می‌توانیم درباره راه‌های همکاری که برای هر دو طرف ارزش ایجاد می‌کند گفتگو کنیم."
	if projValue != "" {
		value = "اگر فرصتی باشد، می‌توانیم درباره «" + truncateStr(projValue, 200) + "» گفتگو کنیم تا ببینیم چطور برای شما ارزش ایجاد می‌کند."
	}
	nextStep := "پیشنهاد می‌کنم در صورت تمایل، یک زمان کوتاه برای ادامه گفتگو یا ارسال اطلاعات تکمیلی هماهنگ کنیم."

	discovery := []string{
		"در حال حاضر مجموعه شما بیشتر روی چه حوزه‌ای فعالیت می‌کند؟",
		"در بخش تأمین یا فروش، بزرگ‌ترین چالشی که دارید چیست؟",
		"آیا الان زمان مناسبی برای صحبت در این باره است؟",
	}
	if projPain != "" {
		discovery = append(discovery, "در «"+truncateStr(projPain, 180)+"» آیا تجربه یا چالش مشخصی داشته‌اید؟")
	}

	objections := []string{
		"الان وقت ندارم.",
		"در حال حاضر به خدمات جدید نیاز نداریم.",
		"هم‌اکنون با تأمین‌کننده دیگری کار می‌کنیم.",
	}
	responses := []string{
		"حتماً، وقت شما را نمی‌گیرم. ممکن است زمان کوتاهی در اختیارم بگذارید یا شماره تماس دیگری معرفی کنید؟",
		"متوجه می‌شوم. اگر در آینده شرایط تغییر کرد، ما آماده همکاری هستیم.",
		"مشکلی نیست. شاید بتوانیم در آینده با هم آشنا شویم؛ اجازه می‌دهید اطلاعات کوتاهی برایتان ارسال کنم؟",
	}
	if projObjections != "" {
		objections = append(objections, truncateStr(projObjections, 160))
		responses = append(responses, "اگر این نگرانی وجود دارد، می‌توانیم در یک گفتگوی کوتاه بررسی کنیم که آیا این موضوع برای شما واقعاً مطرح است یا نه.")
	}

	contextUsed := ContextUsed{Project: projTitle, Lead: company}
	if contextUsed.Lead == "" {
		contextUsed.Lead = display
	}

	leadSummary := target
	if leadCtx != nil && leadCtx.Industry != "" {
		leadSummary = target + " (" + leadCtx.Industry + ")"
	}

	return &CallScenario{
		OpeningSentence:     opening,
		Introduction:        intro,
		ReasonForCall:       reason,
		ValueProposition:    value,
		DiscoveryQuestions:  discovery,
		LikelyObjections:    objections,
		ObjectionResponses:  responses,
		RecommendedNextStep: nextStep,
		ClosingLine:         "ممنون از وقتی که گذاشتید. روز خوبی داشته باشید.",
		ContextUsed:         contextUsed,
		Strategy: &CallScenarioStrategy{
			LeadSummary:        leadSummary,
			Fit:                "uncertain",
			FitSummary:         "not scored — this is a deterministic fallback generated because the LLM was unavailable.",
			PrimaryAngle:       truncateStr(projValue, 140),
			SecondaryAngle:     "",
			LikelyPain:         truncateStr(projPain, 160),
			LikelyPainBasis:    "hypothesis",
			ValidationRequired: true,
			KnownPains:         []string{},
			PainHypotheses:     []string{},
			Uncertainties: []string{
				"the lead's actual needs, workflow and pain",
				"whether the lead has any prior exposure to this project's offer",
			},
			CallObjective: "introduce cooperation and schedule a short follow-up",
			Tone:          "polite, concise, spoken Persian",
		},
		DiscoveryPurposes: []DiscoveryQuestion{
			{Question: discovery[0], Purpose: "understand the lead's current business focus"},
			{Question: discovery[1], Purpose: "surface the lead's biggest operational or sales challenge"},
			{Question: discovery[2], Purpose: "check timing and permission to continue"},
		},
	}
}
