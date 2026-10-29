-- 004_call_scenarios.sql
-- Call scenario persistence per lead with full history, generation indicator,
-- and live teleprompter support.

CREATE TABLE IF NOT EXISTS call_scenarios (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id             UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    contact_id          UUID REFERENCES contacts(id) ON DELETE SET NULL,
    user_id             UUID REFERENCES users(id) ON DELETE SET NULL,
    opening_sentence    TEXT NOT NULL DEFAULT '',
    introduction        TEXT NOT NULL DEFAULT '',
    reason_for_call     TEXT NOT NULL DEFAULT '',
    value_proposition   TEXT NOT NULL DEFAULT '',
    discovery_questions JSONB NOT NULL DEFAULT '[]',
    likely_objections   JSONB NOT NULL DEFAULT '[]',
    objection_responses JSONB NOT NULL DEFAULT '[]',
    recommended_next_step TEXT NOT NULL DEFAULT '',
    closing_line        TEXT NOT NULL DEFAULT '',
    is_generated        BOOLEAN NOT NULL DEFAULT false,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_call_scenarios_lead ON call_scenarios (lead_id, created_at DESC);
