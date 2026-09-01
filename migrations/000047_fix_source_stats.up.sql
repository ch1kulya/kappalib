WITH actual_stats AS (
    SELECT
        source_id,
        COUNT(DISTINCT novel_id) AS actual_novels_count
    FROM chapters
    WHERE source_id IS NOT NULL
    GROUP BY source_id
)
UPDATE sources s
SET novels_count = COALESCE(stats.actual_novels_count, 0)
FROM actual_stats stats
WHERE s.id = stats.source_id;

UPDATE sources
SET novels_count = 0
WHERE id NOT IN (
    SELECT DISTINCT source_id 
    FROM chapters 
    WHERE source_id IS NOT NULL
);

DROP TRIGGER IF EXISTS update_source_chapter_chapter_stats ON chapters;
DROP TRIGGER IF EXISTS trg_update_source_chapter_stats ON chapters;
DROP TRIGGER IF EXISTS trg_chapter_stats_ai ON chapters;
DROP TRIGGER IF EXISTS trg_chapter_stats_ad ON chapters;
DROP TRIGGER IF EXISTS trg_chapter_stats_au ON chapters;

CREATE OR REPLACE FUNCTION update_source_chapter_chapter_stats()
RETURNS TRIGGER AS $$ BEGIN
    IF TG_OP = 'INSERT' THEN
        UPDATE sources s
        SET novels_count = COALESCE(d.actual_count, 0)
        FROM (
            SELECT c.source_id, COUNT(DISTINCT c.novel_id) AS actual_count
            FROM chapters c
            WHERE c.source_id IN (SELECT source_id FROM new_table)
            GROUP BY c.source_id
        ) d
        WHERE s.id = d.source_id;

    ELSIF TG_OP = 'DELETE' THEN
        UPDATE sources s
        SET novels_count = COALESCE(d.actual_count, 0)
        FROM (
            SELECT c.source_id, COUNT(DISTINCT c.novel_id) AS actual_count
            FROM chapters c
            WHERE c.source_id IN (SELECT source_id FROM old_table)
            GROUP BY c.source_id
        ) d
        WHERE s.id = d.source_id;

        UPDATE sources s
        SET novels_count = 0
        WHERE s.id IN (SELECT source_id FROM old_table)
        AND s.id NOT IN (
            SELECT DISTINCT source_id FROM chapters WHERE source_id IS NOT NULL
        );

    ELSIF TG_OP = 'UPDATE' THEN
        UPDATE sources s
        SET novels_count = COALESCE(d.actual_count, 0)
        FROM (
            SELECT c.source_id, COUNT(DISTINCT c.novel_id) AS actual_count
            FROM chapters c
            WHERE c.source_id IN (
                SELECT source_id FROM old_table
                UNION
                SELECT source_id FROM new_table
            )
            GROUP BY c.source_id
        ) d
        WHERE s.id = d.source_id;

        UPDATE sources s
        SET novels_count = 0
        WHERE s.id IN (
            SELECT source_id FROM old_table
            UNION
            SELECT source_id FROM new_table
        )
        AND s.id NOT IN (
            SELECT DISTINCT source_id FROM chapters WHERE source_id IS NOT NULL
        );
    END IF;

    RETURN NULL;
END;
 $$ LANGUAGE plpgsql;

CREATE TRIGGER trg_chapter_stats_ai
    AFTER INSERT ON chapters
    REFERENCING NEW TABLE AS new_table
    FOR EACH STATEMENT
    EXECUTE FUNCTION update_source_chapter_chapter_stats();

CREATE TRIGGER trg_chapter_stats_ad
    AFTER DELETE ON chapters
    REFERENCING OLD TABLE AS old_table
    FOR EACH STATEMENT
    EXECUTE FUNCTION update_source_chapter_chapter_stats();

CREATE TRIGGER trg_chapter_stats_au
    AFTER UPDATE ON chapters
    REFERENCING NEW TABLE AS new_table OLD TABLE AS old_table
    FOR EACH STATEMENT
    EXECUTE FUNCTION update_source_chapter_chapter_stats();