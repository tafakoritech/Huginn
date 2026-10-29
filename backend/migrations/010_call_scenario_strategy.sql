-- 010_call_scenario_strategy.sql
-- Deep lead-specific call scenarios carry a structured sales strategy and rich
-- conversation elements alongside the legacy spoken-script fields.
--
-- strategy: concise lead↔project fit analysis conclusions that drove the
--           script (lead_summary, fit, primary_angle, likely_pain, ...).
-- discovery_question_purposes: each discovery question + the strategic purpose
--           it serves for THIS lead.
-- conversation_branches: contingency strategies for likely call outcomes.

ALTER TABLE call_scenarios ADD COLUMN IF NOT EXISTS strategy JSONB NOT NULL DEFAULT '{}';
ALTER TABLE call_scenarios ADD COLUMN IF NOT EXISTS discovery_question_purposes JSONB NOT NULL DEFAULT '[]';
ALTER TABLE call_scenarios ADD COLUMN IF NOT EXISTS conversation_branches JSONB NOT NULL DEFAULT '[]';
