-- 005_projects.sql
-- Project entity: the sales-initiative / business-context that scopes leads and
-- drives AI scenario generation. Projects are referenced by leads so a single
-- initiative (product, campaign, service, objective) can group many leads.
CREATE TABLE IF NOT EXISTS projects (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title               TEXT NOT NULL,
    description         TEXT NOT NULL,
    product_name        TEXT,
    product_description TEXT,
    target_market       TEXT,
    ideal_customer_profile TEXT,
    customer_pain_points TEXT,
    value_proposition   TEXT,
    sales_objective     TEXT,
    common_objections   TEXT,
    created_by          UUID REFERENCES users(id),
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_projects_created_at ON projects (created_at);
CREATE INDEX IF NOT EXISTS idx_projects_created_by ON projects (created_by);
