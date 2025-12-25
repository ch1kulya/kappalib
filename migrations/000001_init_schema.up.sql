CREATE EXTENSION IF NOT EXISTS pgcrypto;
CREATE EXTENSION IF NOT EXISTS pg_trgm;

CREATE OR REPLACE FUNCTION generate_short_id(prefix TEXT)
RETURNS TEXT AS $$
DECLARE
    chars TEXT := 'abcdefghijklmnopqrstuvwxyz0123456789';
    result TEXT := prefix;
    i INTEGER;
BEGIN
    FOR i IN 1..8 LOOP
        result := result || substr(chars, floor(random() * length(chars) + 1)::integer, 1);
    END LOOP;
    RETURN result;
END;
$$ LANGUAGE plpgsql;

CREATE TABLE IF NOT EXISTS novels (
    id VARCHAR(20) PRIMARY KEY DEFAULT generate_short_id('nvl_'),
    title VARCHAR(500) NOT NULL,
    title_en VARCHAR(500) NOT NULL,
    author VARCHAR(300) NOT NULL,
    year_start INTEGER NOT NULL,
    year_end INTEGER,
    status VARCHAR(50) NOT NULL CHECK (status IN ('ongoing', 'completed', 'announced')),
    description TEXT,
    age_rating VARCHAR(10),
    cover_url TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS chapters (
    id VARCHAR(20) PRIMARY KEY DEFAULT generate_short_id('chp_'),
    novel_id VARCHAR(20) NOT NULL REFERENCES novels(id) ON DELETE CASCADE,
    chapter_num INTEGER NOT NULL,
    title VARCHAR(500) NOT NULL,
    title_en VARCHAR(500),
    content TEXT NOT NULL,
    source VARCHAR(500),
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    UNIQUE(novel_id, chapter_num)
);

CREATE INDEX IF NOT EXISTS idx_chapters_novel_id ON chapters(novel_id);
CREATE INDEX IF NOT EXISTS idx_chapters_novel_chapter ON chapters(novel_id, chapter_num);
CREATE INDEX IF NOT EXISTS idx_novels_created_at ON novels(created_at DESC);

CREATE INDEX IF NOT EXISTS idx_novels_title_trgm ON novels USING gin (title gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_novels_title_en_trgm ON novels USING gin (title_en gin_trgm_ops);
CREATE INDEX IF NOT EXISTS idx_novels_author_trgm ON novels USING gin (author gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_novels_title_norm_trgm_gin
    ON novels USING gin (lower(regexp_replace(title, '[^a-zA-Z0-9]', '', 'g')) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_novels_title_en_norm_trgm_gin
    ON novels USING gin (lower(regexp_replace(title_en, '[^a-zA-Z0-9]', '', 'g')) gin_trgm_ops);

CREATE INDEX IF NOT EXISTS idx_novels_author_norm_trgm_gin
    ON novels USING gin (lower(regexp_replace(author, '[^a-zA-Z0-9]', '', 'g')) gin_trgm_ops);
