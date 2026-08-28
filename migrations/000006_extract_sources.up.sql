CREATE TABLE IF NOT EXISTS sources (
    id serial PRIMARY KEY,
    name varchar(200) NOT NULL UNIQUE,
    logo_url text
);

INSERT INTO sources (name)
SELECT DISTINCT
    source
FROM
    chapters
WHERE
    source IS NOT NULL
    AND source != ''
ON CONFLICT (name)
    DO NOTHING;

ALTER TABLE chapters
    ADD COLUMN IF NOT EXISTS source_id integer REFERENCES sources (id) ON DELETE SET NULL;

UPDATE
    chapters c
SET
    source_id = s.id
FROM
    sources s
WHERE
    c.source = s.name;

ALTER TABLE chapters
    DROP COLUMN IF EXISTS source;

CREATE INDEX IF NOT EXISTS idx_chapters_source_id ON chapters (source_id);

