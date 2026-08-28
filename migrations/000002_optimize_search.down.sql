DROP INDEX IF EXISTS idx_novels_trgm_search;

ALTER TABLE novels
    DROP COLUMN IF EXISTS title_norm,
    DROP COLUMN IF EXISTS title_en_norm,
    DROP COLUMN IF EXISTS author_norm;

CREATE INDEX IF NOT EXISTS idx_novels_title_norm_trgm_gin ON novels USING gin (lower(regexp_replace(title, '[^a-zA-Z0-9]', '', 'g')) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_novels_title_en_norm_trgm_gin ON novels USING gin (lower(regexp_replace(title_en, '[^a-zA-Z0-9]', '', 'g')) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_novels_author_norm_trgm_gin ON novels USING gin (lower(regexp_replace(author, '[^a-zA-Z0-9]', '', 'g')) gin_trgm_ops);

