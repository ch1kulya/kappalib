DROP TRIGGER IF EXISTS trg_update_novel_chapter_count ON chapters;

DROP FUNCTION IF EXISTS update_novel_chapter_count();

DROP INDEX IF EXISTS idx_novels_chapters_count;

ALTER TABLE novels DROP COLUMN IF EXISTS chapters_count;
