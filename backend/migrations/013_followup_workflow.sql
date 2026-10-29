-- 013_followup_workflow.sql
-- Canonical follow-up model: normalize the status enum and add the columns for
-- the full lifecycle. The enum values are added here; they are USED (data
-- update, default, partial index) in migration 014 because PostgreSQL forbids
-- using a newly-added enum value inside the transaction that added it.

ALTER TYPE followup_status ADD VALUE IF NOT EXISTS 'PENDING';
ALTER TYPE followup_status ADD VALUE IF NOT EXISTS 'SNOOZED';

ALTER TABLE follow_ups ADD COLUMN IF NOT EXISTS contact_id UUID REFERENCES contacts(id);
ALTER TABLE follow_ups ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id);
ALTER TABLE follow_ups ADD COLUMN IF NOT EXISTS campaign_id UUID REFERENCES campaigns(id);
ALTER TABLE follow_ups ADD COLUMN IF NOT EXISTS title TEXT;
ALTER TABLE follow_ups ADD COLUMN IF NOT EXISTS channel TEXT;
ALTER TABLE follow_ups ADD COLUMN IF NOT EXISTS priority TEXT NOT NULL DEFAULT 'MEDIUM';
ALTER TABLE follow_ups ADD COLUMN IF NOT EXISTS snoozed_until TIMESTAMPTZ;
ALTER TABLE follow_ups ADD COLUMN IF NOT EXISTS completion_note TEXT;
ALTER TABLE follow_ups ADD COLUMN IF NOT EXISTS outcome TEXT;
ALTER TABLE follow_ups ADD COLUMN IF NOT EXISTS source TEXT NOT NULL DEFAULT 'MANUAL';
ALTER TABLE follow_ups ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();

CREATE UNIQUE INDEX IF NOT EXISTS uq_follow_ups_interaction ON follow_ups (interaction_id) WHERE interaction_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_follow_ups_assigned_due ON follow_ups (assigned_to, status, due_at);
