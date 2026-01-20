DROP TRIGGER IF EXISTS trg_refresh_chapter_groups ON chapters;
DROP FUNCTION IF EXISTS refresh_chapter_groups();
DROP MATERIALIZED VIEW IF EXISTS chapter_groups;
