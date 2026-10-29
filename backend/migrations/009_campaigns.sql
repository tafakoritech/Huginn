-- 009_campaigns.sql
-- Campaign: a time-bound / strategic sales campaign that groups Projects.
-- A Project may optionally belong to a Campaign (projects.campaign_id). The
-- relationship is preserved when a campaign is deleted (ON DELETE SET NULL).
DO $$
BEGIN
    CREATE TYPE campaign_status AS ENUM ('ACTIVE', 'ARCHIVED');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

CREATE TABLE IF NOT EXISTS campaigns (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title       TEXT NOT NULL,
    description TEXT,
    objective   TEXT,
    status      campaign_status NOT NULL DEFAULT 'ACTIVE',
    start_at    TIMESTAMPTZ,
    end_at      TIMESTAMPTZ,
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_campaigns_status ON campaigns (status);
CREATE INDEX IF NOT EXISTS idx_campaigns_created_at ON campaigns (created_at);
CREATE INDEX IF NOT EXISTS idx_campaigns_created_by ON campaigns (created_by);

-- Projects may optionally belong to a Campaign (nullable so existing projects
-- remain valid).
ALTER TABLE projects ADD COLUMN IF NOT EXISTS campaign_id UUID REFERENCES campaigns(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_projects_campaign ON projects (campaign_id);
