-- 011_call_scenario_generation_status.sql
-- Explicit generation provenance for AI call scenarios.
--
-- The historical bug: a deterministic fallback scenario was persisted with
-- is_generated=true and the API returned HTTP 200, so Android/PWA clients
-- could mistake template output for a personalized LLM generation.
--
-- These columns make fallback usage explicit on every stored scenario:
--   generation_status: 'generated' | 'degraded'
--   fallback_used:     true when the scenario is the deterministic template
--   generation_error:  compact reason when the LLM could not produce output

ALTER TABLE call_scenarios ADD COLUMN IF NOT EXISTS generation_status TEXT NOT NULL DEFAULT '';
ALTER TABLE call_scenarios ADD COLUMN IF NOT EXISTS fallback_used BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE call_scenarios ADD COLUMN IF NOT EXISTS generation_error TEXT;
