DROP INDEX IF EXISTS idx_novels_author_norm_trgm_gin;
DROP INDEX IF EXISTS idx_novels_title_en_norm_trgm_gin;
DROP INDEX IF EXISTS idx_novels_title_norm_trgm_gin;
DROP INDEX IF EXISTS idx_novels_author_trgm;
DROP INDEX IF EXISTS idx_novels_title_en_trgm;
DROP INDEX IF EXISTS idx_novels_title_trgm;
DROP INDEX IF EXISTS idx_novels_created_at;
DROP INDEX IF EXISTS idx_chapters_novel_chapter;
DROP INDEX IF EXISTS idx_chapters_novel_id;

DROP TABLE IF EXISTS chapters;
DROP TABLE IF EXISTS novels;

DROP FUNCTION IF EXISTS generate_short_id(TEXT);

DROP EXTENSION IF EXISTS pg_trgm;
DROP EXTENSION IF EXISTS pgcrypto;
