ALTER TABLE chapters ADD COLUMN IF NOT EXISTS source VARCHAR(500);

UPDATE chapters c
SET source = s.name
FROM sources s
WHERE c.source_id = s.id;

DROP INDEX IF EXISTS idx_chapters_source_id;

ALTER TABLE chapters DROP COLUMN IF EXISTS source_id;

DROP TABLE IF EXISTS sources;
