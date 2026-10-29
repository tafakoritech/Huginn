-- 001_initial_schema.sql
-- Enables pg_trgm for fuzzy duplicate detection.
CREATE EXTENSION IF NOT EXISTS pg_trgm;

-- Enum types
DO $$
BEGIN
    CREATE TYPE user_role AS ENUM ('super_admin', 'sales_development_manager', 'sales_agent', 'viewer');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE batch_status AS ENUM ('UPLOADED', 'QUEUED', 'PROCESSING', 'REVIEW_REQUIRED', 'COMPLETED', 'FAILED');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE source_type AS ENUM ('CSV', 'XLSX', 'PDF', 'DOCX', 'TXT', 'HTML', 'JSON', 'IMAGE', 'RAW_TEXT', 'URL');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE lead_status AS ENUM ('NEW', 'READY', 'ASSIGNED', 'CONTACTING', 'FOLLOW_UP', 'QUALIFIED', 'DISQUALIFIED', 'EXPORTED_TO_MUNINN', 'ARCHIVED');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE quality_status AS ENUM ('RAW', 'NORMALIZED', 'VERIFIED', 'UNVERIFIED');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE channel_type AS ENUM ('CALL', 'SMS', 'WHATSAPP', 'TELEGRAM', 'BALE', 'EITAA', 'RUBIKA', 'EMAIL', 'OTHER');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE direction_type AS ENUM ('INBOUND', 'OUTBOUND');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE action_status AS ENUM ('CREATED', 'OPENED', 'COMPLETED', 'FAILED', 'CANCELLED');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE report_status AS ENUM ('PENDING', 'SUBMITTED', 'VOIDED_BY_MANAGER');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE followup_status AS ENUM ('OPEN', 'COMPLETED', 'CANCELLED', 'OVERDUE');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE dup_status AS ENUM ('POSSIBLE', 'IGNORED', 'MERGED', 'DISTINCT');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE export_status AS ENUM ('PENDING', 'IN_PROGRESS', 'SUCCEEDED', 'FAILED', 'SKIPPED');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

DO $$
BEGIN
    CREATE TYPE outbox_status AS ENUM ('PENDING', 'PROCESSING', 'DONE', 'DEAD');
EXCEPTION WHEN duplicate_object THEN NULL;
END $$;

-- Users
CREATE TABLE IF NOT EXISTS users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name          TEXT NOT NULL,
    username      TEXT NOT NULL UNIQUE,
    email         TEXT UNIQUE,
    mobile        TEXT UNIQUE,
    password_hash TEXT NOT NULL,
    role          user_role NOT NULL DEFAULT 'viewer',
    is_active     BOOLEAN NOT NULL DEFAULT true,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_login_at TIMESTAMPTZ,
    deleted_at    TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_users_role ON users (role);
CREATE INDEX IF NOT EXISTS idx_users_username ON users (username);

-- Sessions / refresh tokens
CREATE TABLE IF NOT EXISTS sessions (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    refresh_hash TEXT NOT NULL,
    device       TEXT,
    ip           TEXT,
    user_agent   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at   TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions (user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_revoked ON sessions (revoked_at) WHERE revoked_at IS NULL;

-- Audit log
CREATE TABLE IF NOT EXISTS audit_log (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_user_id UUID REFERENCES users(id),
    action      TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id   TEXT NOT NULL,
    before_json JSONB,
    after_json  JSONB,
    ip          TEXT,
    user_agent  TEXT,
    request_id  TEXT,
    metadata    JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_audit_entity ON audit_log (entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_audit_actor ON audit_log (actor_user_id);
CREATE INDEX IF NOT EXISTS idx_audit_created ON audit_log (created_at);

-- Import batches
CREATE TABLE IF NOT EXISTS import_batches (
    id                   UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name                 TEXT NOT NULL,
    description          TEXT,
    source_type          source_type NOT NULL DEFAULT 'RAW_TEXT',
    status               batch_status NOT NULL DEFAULT 'UPLOADED',
    created_by           UUID REFERENCES users(id),
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now(),
    processing_started_at TIMESTAMPTZ,
    processing_finished_at TIMESTAMPTZ,
    total_documents      INT NOT NULL DEFAULT 0,
    total_candidates     INT NOT NULL DEFAULT 0,
    created_leads        INT NOT NULL DEFAULT 0,
    duplicate_candidates INT NOT NULL DEFAULT 0,
    failed_items         INT NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_import_batches_status ON import_batches (status);
CREATE INDEX IF NOT EXISTS idx_import_batches_created_by ON import_batches (created_by);

-- Source documents
CREATE TABLE IF NOT EXISTS source_documents (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    batch_id     UUID NOT NULL REFERENCES import_batches(id) ON DELETE CASCADE,
    object_key   TEXT,
    bucket       TEXT,
    mime_type    TEXT,
    original_filename TEXT,
    size_bytes   BIGINT,
    checksum     TEXT,
    status       TEXT NOT NULL DEFAULT 'UPLOADED',
    extracted_text TEXT,
    extraction_error TEXT,
    created_by   UUID REFERENCES users(id),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_source_documents_batch ON source_documents (batch_id);
CREATE INDEX IF NOT EXISTS idx_source_documents_status ON source_documents (status);

-- Leads
CREATE TABLE IF NOT EXISTS leads (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    company_name     TEXT NOT NULL,
    display_name     TEXT,
    business_type    TEXT,
    industry         TEXT,
    description      TEXT,
    province         TEXT,
    city             TEXT,
    address          TEXT,
    website          TEXT,
    instagram        TEXT,
    source_batch_id  UUID REFERENCES import_batches(id),
    source_document_id UUID REFERENCES source_documents(id),
    status           lead_status NOT NULL DEFAULT 'NEW',
    quality_status   quality_status NOT NULL DEFAULT 'RAW',
    score            INT NOT NULL DEFAULT 0,
    owner_user_id    UUID REFERENCES users(id),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    archived_at      TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_leads_status ON leads (status);
CREATE INDEX IF NOT EXISTS idx_leads_owner ON leads (owner_user_id);
CREATE INDEX IF NOT EXISTS idx_leads_score ON leads (score);
CREATE INDEX IF NOT EXISTS idx_leads_batch ON leads (source_batch_id);
CREATE INDEX IF NOT EXISTS idx_leads_doc ON leads (source_document_id);
CREATE INDEX IF NOT EXISTS idx_leads_company_trgm ON leads USING gin (company_name gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_leads_city ON leads (city);
CREATE INDEX IF NOT EXISTS idx_leads_industry ON leads (industry);

-- Lead evidence
CREATE TABLE IF NOT EXISTS lead_evidence (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id           UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    source_document_id UUID REFERENCES source_documents(id),
    field_name        TEXT NOT NULL,
    source_page       INT,
    source_row        INT,
    source_excerpt    TEXT,
    confidence        REAL NOT NULL DEFAULT 0
);

CREATE INDEX IF NOT EXISTS idx_evidence_lead ON lead_evidence (lead_id);

-- Contacts
CREATE TABLE IF NOT EXISTS contacts (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id            UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    first_name         TEXT,
    last_name          TEXT,
    full_name          TEXT,
    job_title          TEXT,
    department         TEXT,
    mobile             TEXT,
    mobile_normalized  TEXT,
    phone              TEXT,
    phone_normalized   TEXT,
    email              TEXT,
    email_normalized   TEXT,
    whatsapp           TEXT,
    telegram           TEXT,
    bale               TEXT,
    eitaa              TEXT,
    rubika             TEXT,
    is_decision_maker  BOOLEAN NOT NULL DEFAULT false,
    is_primary_contact BOOLEAN NOT NULL DEFAULT false,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_contacts_lead ON contacts (lead_id);
CREATE INDEX IF NOT EXISTS idx_contacts_mobile_norm ON contacts (mobile_normalized);
CREATE INDEX IF NOT EXISTS idx_contacts_phone_norm ON contacts (phone_normalized);
CREATE INDEX IF NOT EXISTS idx_contacts_email_norm ON contacts (email_normalized);

-- Possible duplicates
CREATE TABLE IF NOT EXISTS possible_duplicates (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_a_id          UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    lead_b_id          UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    reason             TEXT NOT NULL,
    similarity         REAL NOT NULL DEFAULT 0,
    status             dup_status NOT NULL DEFAULT 'POSSIBLE',
    detected_by        TEXT NOT NULL DEFAULT 'system',
    resolved_by        UUID REFERENCES users(id),
    resolved_at        TIMESTAMPTZ,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (lead_a_id, lead_b_id)
);

CREATE INDEX IF NOT EXISTS idx_possible_dups_status ON possible_duplicates (status);

-- Lead tags
CREATE TABLE IF NOT EXISTS tags (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name       TEXT NOT NULL UNIQUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS lead_tags (
    lead_id UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    tag_id  UUID NOT NULL REFERENCES tags(id) ON DELETE CASCADE,
    source  TEXT NOT NULL DEFAULT 'manual', -- 'manual' or 'llm'
    PRIMARY KEY (lead_id, tag_id)
);

-- Assignments
CREATE TABLE IF NOT EXISTS assignments (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id       UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    assignee_id   UUID NOT NULL REFERENCES users(id),
    assigned_by   UUID REFERENCES users(id),
    method        TEXT NOT NULL DEFAULT 'manual', -- manual | bulk | round_robin | reassignment
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (lead_id, assignee_id, method)
);

CREATE INDEX IF NOT EXISTS idx_assignments_lead ON assignments (lead_id);
CREATE INDEX IF NOT EXISTS idx_assignments_assignee ON assignments (assignee_id);

-- Interactions
CREATE TABLE IF NOT EXISTS interactions (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id            UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    contact_id         UUID REFERENCES contacts(id),
    user_id            UUID NOT NULL REFERENCES users(id),
    channel            channel_type NOT NULL,
    direction          direction_type NOT NULL DEFAULT 'OUTBOUND',
    objective          TEXT,
    started_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at           TIMESTAMPTZ,
    duration_seconds   INT,
    action_status      action_status NOT NULL DEFAULT 'CREATED',
    report_status      report_status NOT NULL DEFAULT 'PENDING',
    suggested_message  TEXT,
    suggested_subject  TEXT,
    final_message_if_known TEXT,
    deep_link          TEXT,
    metadata           JSONB,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_interactions_lead ON interactions (lead_id);
CREATE INDEX IF NOT EXISTS idx_interactions_user ON interactions (user_id);
CREATE INDEX IF NOT EXISTS idx_interactions_report_status ON interactions (report_status) WHERE report_status = 'PENDING';

-- Contact reports
CREATE TABLE IF NOT EXISTS contact_reports (
    id                         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    interaction_id             UUID NOT NULL UNIQUE REFERENCES interactions(id) ON DELETE CASCADE,
    outcome                    TEXT NOT NULL,
    contact_reached            BOOLEAN,
    correct_person_reached     BOOLEAN,
    decision_maker_identified  BOOLEAN,
    decision_maker_name        TEXT,
    decision_maker_role        TEXT,
    interest_level             TEXT,
    need_level                 TEXT,
    estimated_purchase_capacity TEXT,
    purchase_timing            TEXT,
    main_objection             TEXT,
    summary                    TEXT,
    next_action                TEXT,
    follow_up_at               TIMESTAMPTZ,
    qualification_recommendation TEXT,
    submitted_by               UUID REFERENCES users(id),
    manager_override           BOOLEAN NOT NULL DEFAULT false,
    manager_override_reason    TEXT,
    overridden_by              UUID REFERENCES users(id),
    created_at                 TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at                 TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_contact_reports_interaction ON contact_reports (interaction_id);

-- Follow-ups
CREATE TABLE IF NOT EXISTS follow_ups (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id        UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    assigned_to    UUID REFERENCES users(id),
    interaction_id UUID REFERENCES interactions(id),
    due_at         TIMESTAMPTZ NOT NULL,
    type           TEXT NOT NULL DEFAULT 'FOLLOW_UP',
    description    TEXT,
    status         followup_status NOT NULL DEFAULT 'OPEN',
    completed_at   TIMESTAMPTZ,
    created_by     UUID REFERENCES users(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_followups_lead ON follow_ups (lead_id);
CREATE INDEX IF NOT EXISTS idx_followups_status ON follow_ups (status) WHERE status = 'OPEN';
CREATE INDEX IF NOT EXISTS idx_followups_due ON follow_ups (due_at);

-- Notes
CREATE TABLE IF NOT EXISTS notes (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id    UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    author_id  UUID NOT NULL REFERENCES users(id),
    text       TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS note_edits (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    note_id     UUID NOT NULL REFERENCES notes(id) ON DELETE CASCADE,
    old_text    TEXT NOT NULL,
    new_text    TEXT NOT NULL,
    edited_by   UUID NOT NULL REFERENCES users(id),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_notes_lead ON notes (lead_id);

-- Score history
CREATE TABLE IF NOT EXISTS score_history (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id    UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    score      INT NOT NULL,
    components JSONB,
    reason     TEXT,
    created_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_score_history_lead ON score_history (lead_id);

-- Disqualifications
CREATE TABLE IF NOT EXISTS disqualifications (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id         UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    reason          TEXT NOT NULL,
    explanation     TEXT,
    disqualified_by UUID REFERENCES users(id),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_disqualifications_lead ON disqualifications (lead_id);
CREATE INDEX IF NOT EXISTS idx_disqualifications_reason ON disqualifications (reason);

-- LLM runs audit
CREATE TABLE IF NOT EXISTS llm_runs (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    purpose          TEXT NOT NULL,
    model            TEXT NOT NULL,
    prompt_version   TEXT,
    entity_type      TEXT,
    entity_id        TEXT,
    request_hash     TEXT,
    structured_output JSONB,
    latency_ms       INT,
    input_tokens     INT,
    output_tokens    INT,
    status           TEXT NOT NULL,
    error            TEXT,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_llm_runs_entity ON llm_runs (entity_type, entity_id);
CREATE INDEX IF NOT EXISTS idx_llm_runs_purpose ON llm_runs (purpose);

-- Muninn exports
CREATE TABLE IF NOT EXISTS crm_exports (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    huginn_lead_id UUID NOT NULL UNIQUE REFERENCES leads(id) ON DELETE CASCADE,
    muninn_lead_id TEXT,
    status        export_status NOT NULL DEFAULT 'PENDING',
    attempt_count INT NOT NULL DEFAULT 0,
    last_error    TEXT,
    payload       JSONB,
    exported_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_crm_exports_status ON crm_exports (status);

-- Outbox events
CREATE TABLE IF NOT EXISTS outbox_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type TEXT NOT NULL,
    payload    JSONB NOT NULL,
    status     outbox_status NOT NULL DEFAULT 'PENDING',
    attempts   INT NOT NULL DEFAULT 0,
    next_retry_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_outbox_status ON outbox_events (status) WHERE status = 'PENDING';

-- Timeline events
CREATE TABLE IF NOT EXISTS timeline_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id     UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    event_type  TEXT NOT NULL,
    title       TEXT NOT NULL,
    description TEXT,
    actor_user_id UUID REFERENCES users(id),
    payload     JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_timeline_lead ON timeline_events (lead_id, created_at);

-- Lead state transitions history
CREATE TABLE IF NOT EXISTS lead_status_history (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    lead_id    UUID NOT NULL REFERENCES leads(id) ON DELETE CASCADE,
    from_status lead_status,
    to_status  lead_status NOT NULL,
    reason     TEXT,
    changed_by UUID REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_status_history_lead ON lead_status_history (lead_id);
