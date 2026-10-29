-- 002_lead_source.sql
-- Adds a selectable/editable source label to leads (e.g. "لیست تولیدکنندگان").
ALTER TABLE leads ADD COLUMN IF NOT EXISTS source TEXT;
CREATE INDEX IF NOT EXISTS idx_leads_source ON leads (source);
