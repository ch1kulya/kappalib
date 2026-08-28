DROP FUNCTION IF EXISTS resolve_tags (TEXT, TEXT[]);

DROP TRIGGER IF EXISTS trg_update_alt_titles_norm ON novels;

DROP FUNCTION IF EXISTS update_alt_titles_norm ();

DROP INDEX IF EXISTS idx_novels_alt_titles_norm_trgm;

DROP TABLE IF EXISTS novel_tags;

ALTER TABLE novels
    DROP COLUMN IF EXISTS alt_titles_norm,
    DROP COLUMN IF EXISTS alt_titles;

DROP INDEX IF EXISTS idx_tags_name_norm;

DROP TABLE IF EXISTS tags;

