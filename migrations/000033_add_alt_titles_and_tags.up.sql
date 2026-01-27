CREATE TABLE IF NOT EXISTS tags (
    id SERIAL PRIMARY KEY,
    name TEXT NOT NULL UNIQUE,
    name_norm TEXT GENERATED ALWAYS AS (
        lower(regexp_replace(name, '[^a-zA-Zа-яА-ЯёЁ0-9]', '', 'g'))
    ) STORED
);

CREATE INDEX IF NOT EXISTS idx_tags_name_norm ON tags (name_norm);

ALTER TABLE novels
ADD COLUMN IF NOT EXISTS alt_titles JSONB DEFAULT '[]'::JSONB
CHECK (jsonb_typeof(alt_titles) = 'array'),
ADD COLUMN IF NOT EXISTS alt_titles_norm TEXT DEFAULT '';

CREATE TABLE IF NOT EXISTS novel_tags (
    novel_id TEXT NOT NULL REFERENCES novels (id) ON DELETE CASCADE,
    tag_id INTEGER NOT NULL REFERENCES tags (id) ON DELETE CASCADE,
    PRIMARY KEY (novel_id, tag_id)
);

CREATE OR REPLACE FUNCTION update_alt_titles_norm()
RETURNS TRIGGER AS $$
BEGIN
    NEW.alt_titles_norm := lower(regexp_replace(
        COALESCE((SELECT string_agg(value, ' ') FROM jsonb_array_elements_text(NEW.alt_titles)), ''),
        '[^a-zA-Zа-яА-ЯёЁ0-9 ]', '', 'g'
    ));
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_alt_titles_norm
BEFORE INSERT OR UPDATE OF alt_titles ON novels
FOR EACH ROW
EXECUTE FUNCTION update_alt_titles_norm();

CREATE INDEX IF NOT EXISTS idx_novels_alt_titles_norm_trgm
ON novels USING gin (alt_titles_norm gin_trgm_ops);

CREATE OR REPLACE FUNCTION resolve_tags(novel TEXT, tag_names TEXT [])
RETURNS VOID AS $$
DECLARE
    tag_name TEXT;
    tag_id INTEGER;
BEGIN
    IF tag_names IS NULL THEN
        RETURN;
    END IF;
    FOREACH tag_name IN ARRAY tag_names LOOP
        tag_name := trim(tag_name);
        IF tag_name = '' THEN
            CONTINUE;
        END IF;
        SELECT id INTO tag_id FROM tags WHERE name = tag_name;
        IF tag_id IS NULL THEN
            INSERT INTO tags (name) VALUES (tag_name) RETURNING id INTO tag_id;
        END IF;
        INSERT INTO novel_tags (novel_id, tag_id) VALUES (novel, tag_id) ON CONFLICT DO NOTHING;
    END LOOP;
END;
$$ LANGUAGE plpgsql;
