-- 008_call_scenario_generation.sql
-- Unique generation tracking for AI call scenarios.
--
-- Every AI scenario generation gets a unique generation_id so that repeated
-- "Generate Again" requests are stored as separate, auditable records and can
-- never be confused with a cached/previous output. sales_angle records which
-- conversation angle the prompt instructed, and context_used is the LLM's
-- own statement of which project/lead context it actually used (debugging aid).

ALTER TABLE call_scenarios ADD COLUMN IF NOT EXISTS generation_id UUID;
ALTER TABLE call_scenarios ADD COLUMN IF NOT EXISTS sales_angle TEXT NOT NULL DEFAULT '';
ALTER TABLE call_scenarios ADD COLUMN IF NOT EXISTS context_used JSONB NOT NULL DEFAULT '{}';

CREATE INDEX IF NOT EXISTS idx_call_scenarios_generation ON call_scenarios (generation_id);
