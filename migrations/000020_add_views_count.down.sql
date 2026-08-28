DROP INDEX IF EXISTS idx_novels_views_count;

ALTER TABLE novels
    DROP COLUMN IF EXISTS views_count;

