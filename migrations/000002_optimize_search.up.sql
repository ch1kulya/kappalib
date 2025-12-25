DROP INDEX IF EXISTS idx_novels_title_norm_trgm_gin;
DROP INDEX IF EXISTS idx_novels_title_en_norm_trgm_gin;
DROP INDEX IF EXISTS idx_novels_author_norm_trgm_gin;

ALTER TABLE novels
    ADD COLUMN IF NOT EXISTS title_norm TEXT GENERATED ALWAYS AS (lower(regexp_replace(title, '[^a-zA-Z0-9]', '', 'g'))) STORED,
    ADD COLUMN IF NOT EXISTS title_en_norm TEXT GENERATED ALWAYS AS (lower(regexp_replace(title_en, '[^a-zA-Z0-9]', '', 'g'))) STORED,
    ADD COLUMN IF NOT EXISTS author_norm TEXT GENERATED ALWAYS AS (lower(regexp_replace(author, '[^a-zA-Z0-9]', '', 'g'))) STORED;

CREATE INDEX IF NOT EXISTS idx_novels_trgm_search
    ON novels USING gin (title_norm gin_trgm_ops, title_en_norm gin_trgm_ops, author_norm gin_trgm_ops);
