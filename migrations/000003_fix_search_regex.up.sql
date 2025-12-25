DROP INDEX IF EXISTS idx_novels_trgm_search;

ALTER TABLE novels
    DROP COLUMN IF EXISTS title_norm,
    DROP COLUMN IF EXISTS title_en_norm,
    DROP COLUMN IF EXISTS author_norm;

ALTER TABLE novels
    ADD COLUMN title_norm TEXT GENERATED ALWAYS AS (lower(regexp_replace(title, '[^[:alnum:]]', '', 'g'))) STORED,
    ADD COLUMN title_en_norm TEXT GENERATED ALWAYS AS (lower(regexp_replace(title_en, '[^[:alnum:]]', '', 'g'))) STORED,
    ADD COLUMN author_norm TEXT GENERATED ALWAYS AS (lower(regexp_replace(author, '[^[:alnum:]]', '', 'g'))) STORED;

CREATE INDEX IF NOT EXISTS idx_novels_trgm_search
    ON novels USING gin (title_norm gin_trgm_ops, title_en_norm gin_trgm_ops, author_norm gin_trgm_ops);
