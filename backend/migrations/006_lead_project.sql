-- 006_lead_project.sql
-- Links leads to a sales Project (the parent context for AI generation) and
-- adds a lifecycle status to projects so dashboards can show active/archived.
DO $$
BEGIN
    CREATE TYPE project_status AS ENUM ('ACTIVE', 'ARCHIVED');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

ALTER TABLE projects ADD COLUMN IF NOT EXISTS status project_status NOT NULL DEFAULT 'ACTIVE';
CREATE INDEX IF NOT EXISTS idx_projects_status ON projects (status);

ALTER TABLE leads ADD COLUMN IF NOT EXISTS project_id UUID REFERENCES projects(id) ON DELETE SET NULL;
CREATE INDEX IF NOT EXISTS idx_leads_project ON leads (project_id);
