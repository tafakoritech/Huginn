-- 014_followup_status_normalize.sql
-- Now that PENDING/SNOOZED exist in the followup_status enum (added in 013),
-- normalize stored statuses and make PENDING the default. Overdue remains a
-- computed state (status='PENDING' AND due_at < now()); stored OVERDUE/OPEN
-- rows become PENDING.

UPDATE follow_ups SET status = 'PENDING' WHERE status IN ('OPEN', 'OVERDUE');
ALTER TABLE follow_ups ALTER COLUMN status SET DEFAULT 'PENDING';

CREATE INDEX IF NOT EXISTS idx_follow_ups_pending_due ON follow_ups (status, due_at) WHERE status = 'PENDING';
