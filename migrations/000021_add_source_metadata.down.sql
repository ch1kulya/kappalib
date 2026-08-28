DROP TRIGGER IF EXISTS trg_update_source_views ON novels;

DROP TRIGGER IF EXISTS trg_update_source_chapter_stats ON chapters;

DROP FUNCTION IF EXISTS update_source_views_count ();

DROP FUNCTION IF EXISTS update_source_chapter_stats ();

DROP INDEX IF EXISTS idx_sources_views_count;

DROP INDEX IF EXISTS idx_sources_chapters_count;

DROP INDEX IF EXISTS idx_sources_novels_count;

ALTER TABLE sources
    DROP COLUMN IF EXISTS views_count;

ALTER TABLE sources
    DROP COLUMN IF EXISTS chapters_count;

ALTER TABLE sources
    DROP COLUMN IF EXISTS novels_count;

