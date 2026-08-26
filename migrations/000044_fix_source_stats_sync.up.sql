CREATE INDEX IF NOT EXISTS idx_chapters_source_novel ON chapters (source_id, novel_id);

DROP TRIGGER IF EXISTS trg_update_source_chapter_stats ON chapters;

CREATE OR REPLACE FUNCTION update_source_chapter_stats()
RETURNS TRIGGER AS $$
DECLARE
    source_ids INTEGER[];
    character_deltas BIGINT[];
BEGIN
    IF TG_OP = 'INSERT' THEN
        SELECT array_agg(source_id), array_agg(characters)
        INTO source_ids, character_deltas
        FROM (
            SELECT source_id, SUM(LENGTH(content)) AS characters
            FROM new_chapters
            WHERE source_id IS NOT NULL
            GROUP BY source_id
        ) delta;

    ELSIF TG_OP = 'DELETE' THEN
        SELECT array_agg(source_id), array_agg(characters)
        INTO source_ids, character_deltas
        FROM (
            SELECT source_id, -SUM(LENGTH(content)) AS characters
            FROM old_chapters
            WHERE source_id IS NOT NULL
            GROUP BY source_id
        ) delta;

    ELSIF TG_OP = 'UPDATE' THEN
        SELECT array_agg(source_id), array_agg(characters)
        INTO source_ids, character_deltas
        FROM (
            SELECT source_id, SUM(characters) AS characters
            FROM (
                SELECT source_id, -LENGTH(content) AS characters
                FROM old_chapters
                WHERE source_id IS NOT NULL
                UNION ALL
                SELECT source_id, LENGTH(content)
                FROM new_chapters
                WHERE source_id IS NOT NULL
            ) changes
            GROUP BY source_id
        ) delta;
    END IF;

    IF source_ids IS NULL THEN
        RETURN NULL;
    END IF;

    UPDATE sources s SET
        chapters_count = actual.chapters,
        novels_count = actual.novels,
        characters_count = GREATEST(0, s.characters_count + d.characters)
    FROM unnest(source_ids, character_deltas) AS d(source_id, characters)
    CROSS JOIN LATERAL (
        SELECT COUNT(*) AS chapters, COUNT(DISTINCT c.novel_id) AS novels
        FROM chapters c
        WHERE c.source_id = d.source_id
    ) actual
    WHERE s.id = d.source_id;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_source_chapter_stats_insert
AFTER INSERT ON chapters
REFERENCING NEW TABLE AS new_chapters
FOR EACH STATEMENT EXECUTE FUNCTION update_source_chapter_stats();

CREATE TRIGGER trg_source_chapter_stats_update
AFTER UPDATE ON chapters
REFERENCING OLD TABLE AS old_chapters NEW TABLE AS new_chapters
FOR EACH STATEMENT EXECUTE FUNCTION update_source_chapter_stats();

CREATE TRIGGER trg_source_chapter_stats_delete
AFTER DELETE ON chapters
REFERENCING OLD TABLE AS old_chapters
FOR EACH STATEMENT EXECUTE FUNCTION update_source_chapter_stats();

WITH actual AS (
    SELECT s.id, COUNT(c.id) AS chapters, COUNT(DISTINCT c.novel_id) AS novels
    FROM sources s
    LEFT JOIN chapters c ON c.source_id = s.id
    GROUP BY s.id
)
UPDATE sources s SET
    chapters_count = actual.chapters,
    novels_count = actual.novels
FROM actual
WHERE s.id = actual.id
  AND (s.chapters_count <> actual.chapters OR s.novels_count <> actual.novels);
