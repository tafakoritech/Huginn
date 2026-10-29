-- 003_lead_versioning.sql
-- Immutable per-edit snapshots of lead state. Every edit appends a new row,
-- so no historical value is ever deleted or silently overwritten.

-- Each lead tracks the current (latest) version number. The version is
-- bumped transactionally while the lead row is locked (SELECT ... FOR UPDATE),
-- which serializes concurrent edits and prevents version corruption.
ALTER TABLE leads ADD COLUMN IF NOT EXISTS current_version INT NOT NULL DEFAULT 0;

-- Contacts become soft-deletable so removed contacts remain visible in older
-- versions and never vanish from history.
ALTER TABLE contacts ADD COLUMN IF NOT EXISTS deleted_at TIMESTAMPTZ;

CREATE TABLE IF NOT EXISTS lead_versions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id         UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    version         INT NOT NULL,
    data            JSONB NOT NULL,          -- full snapshot (lead + contacts + tags)
    changes         JSONB NOT NULL DEFAULT '[]', -- [{field, old, new}]
    changed_by      UUID REFERENCES users(id),
    changed_by_role TEXT,
    note            TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (lead_id, version)
);

CREATE INDEX IF NOT EXISTS idx_lead_versions_lead ON lead_versions (lead_id, version DESC);
CREATE INDEX IF NOT EXISTS idx_lead_versions_created ON lead_versions (created_at DESC);
