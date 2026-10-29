package callscenarios

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/tafakorpardazan/huginn/internal/database"
)

var ErrNotFound = errors.New("call scenario not found")

// ContextUsed records which project/lead/angle context a generation actually
// used. It is stored for debugging AI scenario generation.
type ContextUsed struct {
	Project    string `json:"project"`
	Lead       string `json:"lead"`
	SalesAngle string `json:"sales_angle"`
}

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
	// LikelyPainBasis classifies the likely_pain statement as "fact" or
	// "hypothesis"; "hypothesis" content must never be spoken as factual.
	LikelyPainBasis string `json:"likely_pain_basis,omitempty"`
	// ValidationRequired is true when LikelyPain is a hypothesis that must be
	// validated via discovery rather than treated as known.
	ValidationRequired bool `json:"validation_required,omitempty"`
	// KnownPains are pains directly supported by provided context.
	KnownPains []string `json:"known_pains,omitempty"`
	// PainHypotheses are plausible pains to be converted into discovery
	// questions, never stated as facts.
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

// CallScenario persists a lead-specific spoken script.
type CallScenario struct {
	ID           string  `json:"id"`
	GenerationID string  `json:"generation_id,omitempty"`
	LeadID       string  `json:"lead_id"`
	ContactID    *string `json:"contact_id,omitempty"`
	UserID       *string `json:"user_id,omitempty"`
	// ProjectID scopes the scenario to the Project whose context produced it.
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
	IsGenerated          bool                  `json:"is_generated"`
	// GenerationStatus is "generated" or "degraded" and makes LLM fallback
	// usage explicit to API clients (see messaging.GenerationStatus*).
	GenerationStatus string `json:"generation_status,omitempty"`
	// FallbackUsed is true when the scenario is the deterministic template.
	// Always serialized: clients must see an explicit false, never an absent
	// field.
	FallbackUsed bool `json:"fallback_used"`
	// GenerationError carries the reason when the LLM could not produce the
	// scenario (empty when generation succeeded).
	GenerationError string    `json:"generation_error,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type Repository struct {
	db *database.DB
}

// scenarioCols is the canonical SELECT column list for call_scenarios. All
// queries (Create RETURNING, Get, LatestForLead, ListForLead) and scanScenario
// MUST use exactly this list so adding a column cannot silently desync a scan.
const scenarioCols = `id, lead_id, contact_id, user_id, project_id, generation_id, sales_angle,
	opening_sentence, introduction, reason_for_call, value_proposition,
	discovery_questions, likely_objections, objection_responses,
	recommended_next_step, closing_line, context_used, strategy,
	discovery_question_purposes, conversation_branches, is_generated,
	generation_status, fallback_used, generation_error, created_at, updated_at`

func NewRepository(db *database.DB) *Repository {
	return &Repository{db: db}
}

func (r *Repository) Create(ctx context.Context, s *CallScenario) (*CallScenario, error) {
	discQ, _ := json.Marshal(s.DiscoveryQuestions)
	if s.DiscoveryQuestions == nil {
		discQ = []byte("[]")
	}
	objQ, _ := json.Marshal(s.LikelyObjections)
	if s.LikelyObjections == nil {
		objQ = []byte("[]")
	}
	respQ, _ := json.Marshal(s.ObjectionResponses)
	if s.ObjectionResponses == nil {
		respQ = []byte("[]")
	}
	ctxQ, _ := json.Marshal(s.ContextUsed)
	strategyQ, _ := json.Marshal(s.Strategy)
	if s.Strategy == nil {
		strategyQ = []byte("{}")
	}
	purposesQ, _ := json.Marshal(s.DiscoveryPurposes)
	if s.DiscoveryPurposes == nil {
		purposesQ = []byte("[]")
	}
	branchesQ, _ := json.Marshal(s.ConversationBranches)
	if s.ConversationBranches == nil {
		branchesQ = []byte("[]")
	}

	row := r.db.Pool().QueryRow(ctx, `
		INSERT INTO call_scenarios (
			lead_id, contact_id, user_id, project_id, generation_id, sales_angle,
			opening_sentence, introduction, reason_for_call, value_proposition,
			discovery_questions, likely_objections, objection_responses,
			recommended_next_step, closing_line, context_used, strategy,
			discovery_question_purposes, conversation_branches, is_generated,
			generation_status, fallback_used, generation_error
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,$19,$20,$21,$22,$23)
		RETURNING `+scenarioCols,
		s.LeadID, s.ContactID, s.UserID, s.ProjectID, nullID(s.GenerationID), s.SalesAngle,
		s.OpeningSentence, s.Introduction, s.ReasonForCall, s.ValueProposition,
		discQ, objQ, respQ,
		s.RecommendedNextStep, s.ClosingLine, ctxQ, strategyQ, purposesQ, branchesQ, s.IsGenerated,
		s.GenerationStatus, s.FallbackUsed, nullGenerationError(s.GenerationError),
	)
	return scanScenario(row)
}

func (r *Repository) Get(ctx context.Context, leadID, scenarioID string) (*CallScenario, error) {
	row := r.db.Pool().QueryRow(ctx, `
		SELECT `+scenarioCols+`
		FROM call_scenarios
		WHERE id = $1 AND lead_id = $2`, scenarioID, leadID)
	return scanScenario(row)
}

// LatestForLead returns the most recent scenario for a lead (or ErrNotFound).
func (r *Repository) LatestForLead(ctx context.Context, leadID string) (*CallScenario, error) {
	row := r.db.Pool().QueryRow(ctx, `
		SELECT `+scenarioCols+`
		FROM call_scenarios
		WHERE lead_id = $1
		ORDER BY created_at DESC LIMIT 1`, leadID)
	return scanScenario(row)
}

func (r *Repository) Update(ctx context.Context, s *CallScenario) (*CallScenario, error) {
	discQ, _ := json.Marshal(s.DiscoveryQuestions)
	if s.DiscoveryQuestions == nil {
		discQ = []byte("[]")
	}
	objQ, _ := json.Marshal(s.LikelyObjections)
	if s.LikelyObjections == nil {
		objQ = []byte("[]")
	}
	respQ, _ := json.Marshal(s.ObjectionResponses)
	if s.ObjectionResponses == nil {
		respQ = []byte("[]")
	}

	row := r.db.Pool().QueryRow(ctx, `
		UPDATE call_scenarios SET
			contact_id = COALESCE($3, contact_id),
			project_id = COALESCE($4, project_id),
			opening_sentence = $5,
			introduction = $6,
			reason_for_call = $7,
			value_proposition = $8,
			discovery_questions = $9,
			likely_objections = $10,
			objection_responses = $11,
			recommended_next_step = $12,
			closing_line = $13,
			strategy = COALESCE($14, strategy),
			discovery_question_purposes = COALESCE($15, discovery_question_purposes),
			conversation_branches = COALESCE($16, conversation_branches),
			updated_at = now()
		WHERE id = $1 AND lead_id = $2
		RETURNING `+scenarioCols,
		s.ID, s.LeadID, s.ContactID, s.ProjectID,
		s.OpeningSentence, s.Introduction, s.ReasonForCall, s.ValueProposition,
		discQ, objQ, respQ,
		s.RecommendedNextStep, s.ClosingLine,
		jsonOrNull(s.Strategy), jsonOrNull(s.DiscoveryPurposes), jsonOrNull(s.ConversationBranches),
	)
	return scanScenario(row)
}

// jsonOrNull marshals v as JSON or returns nil when v is nil, so COALESCE keeps
// the existing DB value when the client does not supply the field.
func jsonOrNull(v any) any {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return nil
	}
	return b
}

func (r *Repository) Delete(ctx context.Context, leadID, scenarioID string) error {
	tag, err := r.db.Pool().Exec(ctx, `DELETE FROM call_scenarios WHERE id = $1 AND lead_id = $2`, scenarioID, leadID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repository) ListForLead(ctx context.Context, leadID string) ([]*CallScenario, error) {
	rows, err := r.db.Pool().Query(ctx, `
		SELECT `+scenarioCols+`
		FROM call_scenarios
		WHERE lead_id = $1
		ORDER BY created_at DESC`, leadID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := []*CallScenario{}
	for rows.Next() {
		s, err := scanScenario(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func scanScenario(row database.RowScanner) (*CallScenario, error) {
	var s CallScenario
	var discQBytes, objQBytes, respQBytes, ctxQBytes, strategyQBytes, purposesQBytes, branchesQBytes []byte
	var generationError *string
	err := row.Scan(
		&s.ID,
		&s.LeadID,
		&s.ContactID,
		&s.UserID,
		&s.ProjectID,
		&s.GenerationID,
		&s.SalesAngle,
		&s.OpeningSentence,
		&s.Introduction,
		&s.ReasonForCall,
		&s.ValueProposition,
		&discQBytes,
		&objQBytes,
		&respQBytes,
		&s.RecommendedNextStep,
		&s.ClosingLine,
		&ctxQBytes,
		&strategyQBytes,
		&purposesQBytes,
		&branchesQBytes,
		&s.IsGenerated,
		&s.GenerationStatus,
		&s.FallbackUsed,
		&generationError,
		&s.CreatedAt,
		&s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if generationError != nil {
		s.GenerationError = *generationError
	}
	if len(discQBytes) > 0 {
		_ = json.Unmarshal(discQBytes, &s.DiscoveryQuestions)
	}
	if s.DiscoveryQuestions == nil {
		s.DiscoveryQuestions = []string{}
	}
	if len(objQBytes) > 0 {
		_ = json.Unmarshal(objQBytes, &s.LikelyObjections)
	}
	if s.LikelyObjections == nil {
		s.LikelyObjections = []string{}
	}
	if len(respQBytes) > 0 {
		_ = json.Unmarshal(respQBytes, &s.ObjectionResponses)
	}
	if s.ObjectionResponses == nil {
		s.ObjectionResponses = []string{}
	}
	if len(ctxQBytes) > 0 {
		_ = json.Unmarshal(ctxQBytes, &s.ContextUsed)
	}
	if len(strategyQBytes) > 0 && string(strategyQBytes) != "{}" && string(strategyQBytes) != "null" {
		var st CallScenarioStrategy
		if err := json.Unmarshal(strategyQBytes, &st); err == nil {
			s.Strategy = &st
		}
	}
	if len(purposesQBytes) > 0 {
		_ = json.Unmarshal(purposesQBytes, &s.DiscoveryPurposes)
	}
	if s.DiscoveryPurposes == nil {
		s.DiscoveryPurposes = []DiscoveryQuestion{}
	}
	if len(branchesQBytes) > 0 {
		_ = json.Unmarshal(branchesQBytes, &s.ConversationBranches)
	}
	if s.ConversationBranches == nil {
		s.ConversationBranches = []ConversationBranch{}
	}
	return &s, nil
}

// EnsureSchema applies idempotent schema additions so call-scenario generation
// works even when migrations have not been applied (mirrors other repos).
func (r *Repository) EnsureSchema(ctx context.Context) error {
	execs := []string{
		`ALTER TABLE call_scenarios ADD COLUMN IF NOT EXISTS strategy JSONB NOT NULL DEFAULT '{}'`,
		`ALTER TABLE call_scenarios ADD COLUMN IF NOT EXISTS discovery_question_purposes JSONB NOT NULL DEFAULT '[]'`,
		`ALTER TABLE call_scenarios ADD COLUMN IF NOT EXISTS conversation_branches JSONB NOT NULL DEFAULT '[]'`,
		`ALTER TABLE call_scenarios ADD COLUMN IF NOT EXISTS generation_status TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE call_scenarios ADD COLUMN IF NOT EXISTS fallback_used BOOLEAN NOT NULL DEFAULT false`,
		`ALTER TABLE call_scenarios ADD COLUMN IF NOT EXISTS generation_error TEXT`,
	}
	for _, q := range execs {
		if _, err := r.db.Pool().Exec(ctx, q); err != nil {
			return err
		}
	}
	return nil
}

// nullID maps an empty generation id to NULL so the UUID column stays valid for
// manually created scenarios.
func nullID(id string) any {
	if id == "" {
		return nil
	}
	return id
}

// nullGenerationError maps an empty generation error to NULL so manually
// created scenarios keep a clean row.
func nullGenerationError(s string) any {
	if s == "" {
		return nil
	}
	return s
}
