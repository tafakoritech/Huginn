-- 015_next_actions_indexes.sql
-- Indexes that the Next Best Action / Work Queue engine relies on. The queue is
-- built from follow_ups (assigned_to / status / due_at), interactions
-- (user_id / report_status) and leads (owner_user_id / status / score). These
-- indexes keep the home-screen request a single fast query set.

CREATE INDEX IF NOT EXISTS idx_follow_ups_assigned_due ON follow_ups (assigned_to, due_at) WHERE status IN ('PENDING','SNOOZED');
CREATE INDEX IF NOT EXISTS idx_interactions_user_report ON interactions (user_id, report_status) WHERE report_status = 'PENDING';
CREATE INDEX IF NOT EXISTS idx_interactions_lead_started ON interactions (lead_id, started_at);
CREATE INDEX IF NOT EXISTS idx_leads_owner_status_score ON leads (owner_user_id, status, score) WHERE status IN ('ASSIGNED','CONTACTING','FOLLOW_UP');
CREATE INDEX IF NOT EXISTS idx_project_leads_assigned_user ON project_leads (assigned_user_id) WHERE assigned_user_id IS NOT NULL;
