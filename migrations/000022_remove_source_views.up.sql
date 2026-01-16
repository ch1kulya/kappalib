DROP TRIGGER IF EXISTS trg_update_source_views ON novels;
DROP FUNCTION IF EXISTS update_source_views_count();
DROP INDEX IF EXISTS idx_sources_views_count;
ALTER TABLE sources DROP COLUMN IF EXISTS views_count;
