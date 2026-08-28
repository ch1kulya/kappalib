CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE OR REPLACE FUNCTION generate_short_id (prefix text)
    RETURNS text
    AS $$
DECLARE
    chars text := 'abcdefghijklmnopqrstuvwxyz0123456789';
    result text := prefix;
    i integer;
BEGIN
    FOR i IN 1..8 LOOP
        result := result || substr(chars, floor(random() * length(chars) + 1)::integer, 1);
    END LOOP;
    RETURN result;
END;
$$
LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS novels (
    id varchar(20) PRIMARY KEY DEFAULT generate_short_id ('nvl_'),
    title varchar(500) NOT NULL,
    title_en varchar(500) NOT NULL,
    author varchar(300) NOT NULL,
    year_start integer NOT NULL,
    year_end integer,
    status varchar(50) NOT NULL CHECK (status IN ('ongoing', 'completed', 'announced')),
    description text,
    age_rating varchar(10),
    cover_url text,
    created_at timestamp DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chapters (
    id varchar(20) PRIMARY KEY DEFAULT generate_short_id ('chp_'),
    novel_id varchar(20) NOT NULL REFERENCES novels (id) ON DELETE CASCADE,
    chapter_num integer NOT NULL,
    title varchar(500) NOT NULL,
    title_en varchar(500),
    content text NOT NULL,
    source varchar(500),
    created_at timestamp DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (novel_id, chapter_num)
);

CREATE INDEX IF NOT EXISTS idx_chapters_novel_id ON chapters (novel_id);

CREATE INDEX IF NOT EXISTS idx_chapters_novel_chapter ON chapters (novel_id, chapter_num);

CREATE INDEX IF NOT EXISTS idx_novels_created_at ON novels (created_at DESC);

CREATE INDEX IF NOT EXISTS idx_novels_title_trgm ON novels USING gin (title gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_novels_title_en_trgm ON novels USING gin (title_en gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_novels_author_trgm ON novels USING gin (author gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_novels_title_norm_trgm_gin ON novels USING gin (lower(regexp_replace(title, '[^a-zA-Z0-9]', '', 'g')) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_novels_title_en_norm_trgm_gin ON novels USING gin (lower(regexp_replace(title_en, '[^a-zA-Z0-9]', '', 'g')) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_novels_author_norm_trgm_gin ON novels USING gin (lower(regexp_replace(author, '[^a-zA-Z0-9]', '', 'g')) gin_trgm_ops);

