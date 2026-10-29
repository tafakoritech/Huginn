-- 007_import_batch_project.sql
-- Import batches may carry a project so every lead extracted from a batch
-- inherits the chosen project as its AI sales context.
ALTER TABLE import_batches ADD COLUMN IF NOT EXISTS project_id UUID;
CREATE INDEX IF NOT EXISTS idx_import_batches_project ON import_batches (project_id);