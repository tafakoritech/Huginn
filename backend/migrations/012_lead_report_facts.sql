-- 012_lead_report_facts.sql
-- Canonical persistent facts learned from call reports.
--
-- Call reports remain historical snapshots; the facts below are the CURRENT
-- best-known values about the lead, promoted from the most recent report that
-- established them. They let the app pre-fill the next mandatory report so the
-- salesperson is never asked to re-enter stable information (decision maker,
-- correct person) on every call.

ALTER TABLE leads ADD COLUMN IF NOT EXISTS correct_person_reached BOOLEAN;
ALTER TABLE leads ADD COLUMN IF NOT EXISTS decision_maker_identified BOOLEAN NOT NULL DEFAULT false;
ALTER TABLE leads ADD COLUMN IF NOT EXISTS decision_maker_name TEXT;
ALTER TABLE leads ADD COLUMN IF NOT EXISTS decision_maker_role TEXT;

CREATE INDEX IF NOT EXISTS idx_leads_decision_maker ON leads (decision_maker_identified) WHERE decision_maker_identified = true;
