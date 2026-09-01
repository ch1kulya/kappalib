ALTER TABLE sources
    ADD COLUMN IF NOT EXISTS characters_count bigint NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_sources_characters_count ON sources (characters_count DESC);

WITH stats AS (
    SELECT
        source_id,
        COUNT(*) AS chapters,
        COUNT(DISTINCT novel_id) AS novels,
        COALESCE(SUM(LENGTH(content)), 0) AS characters
    FROM
        chapters
    WHERE
        source_id IS NOT NULL
    GROUP BY
        source_id)
UPDATE
    sources s
SET
    chapters_count = COALESCE(stats.chapters, 0),
    novels_count = COALESCE(stats.novels, 0),
    characters_count = COALESCE(stats.characters, 0)
FROM
    stats
WHERE
    s.id = stats.source_id;

UPDATE
    sources
SET
    chapters_count = 0,
    novels_count = 0,
    characters_count = 0
WHERE
    id NOT IN ( SELECT DISTINCT
            source_id
        FROM
            chapters
        WHERE
            source_id IS NOT NULL);

CREATE OR REPLACE FUNCTION update_source_chapter_stats ()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        WITH new_pairs AS (
            SELECT source_id, novel_id, COUNT(*) AS added_chapters, COALESCE(SUM(LENGTH(content)),0) AS added_chars
            FROM new_rows
            WHERE source_id IS NOT NULL
            GROUP BY source_id, novel_id
        ),
        source_deltas AS (
            SELECT source_id, SUM(added_chapters) AS chapters_delta, SUM(added_chars) AS characters_delta
            FROM new_pairs GROUP BY source_id
        ),
        novels_delta AS (
            SELECT np.source_id, COUNT(*) AS novels_delta
            FROM new_pairs np
            WHERE (SELECT COALESCE(COUNT(*),0) FROM chapters c WHERE c.source_id = np.source_id AND c.novel_id = np.novel_id) - np.added_chapters = 0
            GROUP BY np.source_id
        )
        UPDATE sources s
        SET chapters_count = s.chapters_count + sd.chapters_delta,
            characters_count = s.characters_count + sd.characters_delta
        FROM source_deltas sd
        WHERE s.id = sd.source_id;

        UPDATE sources s
        SET novels_count = s.novels_count + nd.novels_delta
        FROM novels_delta nd
        WHERE s.id = nd.source_id;

        RETURN NULL;

    ELSIF TG_OP = 'DELETE' THEN
        WITH old_pairs AS (
            SELECT source_id, novel_id, COUNT(*) AS rem_chapters, COALESCE(SUM(LENGTH(content)),0) AS rem_chars
            FROM old_rows
            WHERE source_id IS NOT NULL
            GROUP BY source_id, novel_id
        ),
        source_deltas AS (
            SELECT source_id, SUM(rem_chapters) AS chapters_delta, SUM(rem_chars) AS characters_delta
            FROM old_pairs GROUP BY source_id
        ),
        novels_delta AS (
            SELECT op.source_id, COUNT(*) AS novels_delta
            FROM old_pairs op
            WHERE (SELECT COALESCE(COUNT(*),0) FROM chapters c WHERE c.source_id = op.source_id AND c.novel_id = op.novel_id) = 0
            GROUP BY op.source_id
        )
        UPDATE sources s
        SET chapters_count = GREATEST(0, s.chapters_count - sd.chapters_delta),
            characters_count = GREATEST(0, s.characters_count - sd.characters_delta)
        FROM source_deltas sd
        WHERE s.id = sd.source_id;

        UPDATE sources s
        SET novels_count = GREATEST(0, s.novels_count - nd.novels_delta)
        FROM novels_delta nd
        WHERE s.id = nd.source_id;

        RETURN NULL;

    ELSIF TG_OP = 'UPDATE' THEN
        WITH affected AS (
            SELECT DISTINCT source_id FROM old_rows WHERE source_id IS NOT NULL
            UNION
            SELECT DISTINCT source_id FROM new_rows WHERE source_id IS NOT NULL
        ),
        agg AS (
            SELECT s2.id AS source_id,
                   COUNT(c.*) AS chapters,
                   COUNT(DISTINCT c.novel_id) AS novels,
                   COALESCE(SUM(LENGTH(c.content)),0) AS characters
            FROM sources s2
            LEFT JOIN chapters c ON c.source_id = s2.id
            WHERE s2.id IN (SELECT source_id FROM affected)
            GROUP BY s2.id
        )
        UPDATE sources s
        SET chapters_count = COALESCE(a.chapters, 0),
            novels_count = COALESCE(a.novels, 0),
            characters_count = COALESCE(a.characters, 0)
        FROM agg a
        WHERE s.id = a.source_id;

        RETURN NULL;
    END IF;

    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_update_source_chapter_stats ON chapters;
DROP TRIGGER IF EXISTS trg_update_source_chapter_stats_insert ON chapters;
DROP TRIGGER IF EXISTS trg_update_source_chapter_stats_delete ON chapters;
DROP TRIGGER IF EXISTS trg_update_source_chapter_stats_update ON chapters;

CREATE TRIGGER trg_update_source_chapter_stats_insert
    AFTER INSERT ON chapters
    REFERENCING NEW TABLE AS new_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION update_source_chapter_stats();

CREATE TRIGGER trg_update_source_chapter_stats_delete
    AFTER DELETE ON chapters
    REFERENCING OLD TABLE AS old_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION update_source_chapter_stats();

CREATE TRIGGER trg_update_source_chapter_stats_update
    AFTER UPDATE ON chapters
    REFERENCING NEW TABLE AS new_rows OLD TABLE AS old_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION update_source_chapter_stats();