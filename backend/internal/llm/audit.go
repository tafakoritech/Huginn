package llm

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
	"github.com/tafakorpardazan/huginn/internal/database"
)

// PostgresAuditor persists LLM run records to the llm_runs table.
type PostgresAuditor struct {
	db *database.DB
}

func NewPostgresAuditor(db *database.DB) *PostgresAuditor { return &PostgresAuditor{db: db} }

func (a *PostgresAuditor) Record(ctx context.Context, run *Run) error {
	if run.ID == "" {
		run.ID = uuid.NewString()
	}
	var structured any
	if len(run.StructuredOutput) > 0 {
		_ = json.Unmarshal(run.StructuredOutput, &structured)
	}
	_, err := a.db.Pool().Exec(ctx, `
		INSERT INTO llm_runs (id, purpose, model, prompt_version, entity_type, entity_id, request_hash, structured_output, latency_ms, input_tokens, output_tokens, status, error)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13)`,
		run.ID, run.Purpose, run.Model, run.PromptVersion, run.EntityType, run.EntityID,
		run.RequestHash, structured, run.LatencyMs, run.InputTokens, run.OutputTokens, run.Status, nullIfEmpty(run.Error))
	return err
}

func nullIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}
