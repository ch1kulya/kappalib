CREATE INDEX IF NOT EXISTS idx_chapters_novel_id_source_id ON chapters (novel_id, source_id, id);

DROP TRIGGER IF EXISTS trg_update_source_chapter_stats ON chapters;

DROP FUNCTION IF EXISTS update_source_chapter_stats ();

CREATE OR REPLACE FUNCTION update_source_chapter_stats_insert ()
    RETURNS TRIGGER
    AS $$
BEGIN
    WITH batch AS (
        SELECT
            source_id,
            COUNT(*) AS chapters_delta,
            COALESCE(SUM(LENGTH(content)), 0) AS chars_delta
        FROM
            new_rows
        WHERE
            source_id IS NOT NULL
        GROUP BY
            source_id)
    UPDATE
        sources s
    SET
        chapters_count = s.chapters_count + b.chapters_delta,
        characters_count = s.characters_count + b.chars_delta
    FROM
        batch b
    WHERE
        s.id = b.source_id;
    WITH pairs AS (
        SELECT DISTINCT
            source_id,
            novel_id
        FROM
            new_rows
        WHERE
            source_id IS NOT NULL
),
new_pairs AS (
    SELECT
        p.source_id
    FROM
        pairs p
    WHERE
        NOT EXISTS (
            SELECT
                1
            FROM
                chapters c
            WHERE
                c.novel_id = p.novel_id
                AND c.source_id = p.source_id
                AND NOT EXISTS (
                    SELECT
                        1
                    FROM
                        new_rows nr
                    WHERE
                        nr.id = c.id))
),
per_source AS (
    SELECT
        source_id,
        COUNT(*) AS novels_delta
    FROM
        new_pairs
    GROUP BY
        source_id)
UPDATE
    sources s
SET
    novels_count = s.novels_count + ps.novels_delta
FROM
    per_source ps
WHERE
    s.id = ps.source_id;
    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION update_source_chapter_stats_delete ()
    RETURNS TRIGGER
    AS $$
BEGIN
    WITH batch AS (
        SELECT
            source_id,
            COUNT(*) AS chapters_delta,
            COALESCE(SUM(LENGTH(content)), 0) AS chars_delta
        FROM
            old_rows
        WHERE
            source_id IS NOT NULL
        GROUP BY
            source_id)
    UPDATE
        sources s
    SET
        chapters_count = GREATEST (0, s.chapters_count - b.chapters_delta),
        characters_count = GREATEST (0, s.characters_count - b.chars_delta)
    FROM
        batch b
    WHERE
        s.id = b.source_id;
    WITH pairs AS (
        SELECT DISTINCT
            source_id,
            novel_id
        FROM
            old_rows
        WHERE
            source_id IS NOT NULL
),
gone_pairs AS (
    SELECT
        p.source_id
    FROM
        pairs p
    WHERE
        NOT EXISTS (
            SELECT
                1
            FROM
                chapters c
            WHERE
                c.novel_id = p.novel_id
                AND c.source_id = p.source_id)
),
per_source AS (
    SELECT
        source_id,
        COUNT(*) AS novels_delta
    FROM
        gone_pairs
    GROUP BY
        source_id)
UPDATE
    sources s
SET
    novels_count = GREATEST (0, s.novels_count - ps.novels_delta)
FROM
    per_source ps
WHERE
    s.id = ps.source_id;
    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE FUNCTION update_source_chapter_stats_update ()
    RETURNS TRIGGER
    AS $$
BEGIN
    WITH old_side AS (
        SELECT
            source_id,
            COUNT(*) AS chapters_delta,
            COALESCE(SUM(LENGTH(content)), 0) AS chars_delta
        FROM
            old_rows
        WHERE
            source_id IS NOT NULL
        GROUP BY
            source_id
),
new_side AS (
    SELECT
        source_id,
        COUNT(*) AS chapters_delta,
        COALESCE(SUM(LENGTH(content)), 0) AS chars_delta
    FROM
        new_rows
    WHERE
        source_id IS NOT NULL
    GROUP BY
        source_id
),
deltas AS (
    SELECT
        COALESCE(o.source_id, n.source_id) AS source_id,
        COALESCE(n.chapters_delta, 0) - COALESCE(o.chapters_delta, 0) AS chapters_delta,
        COALESCE(n.chars_delta, 0) - COALESCE(o.chars_delta, 0) AS chars_delta
    FROM
        old_side o
        FULL JOIN new_side n ON n.source_id = o.source_id)
UPDATE
    sources s
SET
    chapters_count = GREATEST (0, s.chapters_count + d.chapters_delta),
    characters_count = GREATEST (0, s.characters_count + d.chars_delta)
FROM
    deltas d
WHERE
    s.id = d.source_id;
    WITH pairs AS (
        SELECT DISTINCT
            source_id,
            novel_id
        FROM
            old_rows
        WHERE
            source_id IS NOT NULL
),
gone_pairs AS (
    SELECT
        p.source_id
    FROM
        pairs p
    WHERE
        NOT EXISTS (
            SELECT
                1
            FROM
                chapters c
            WHERE
                c.novel_id = p.novel_id
                AND c.source_id = p.source_id)
),
per_source AS (
    SELECT
        source_id,
        COUNT(*) AS novels_delta
    FROM
        gone_pairs
    GROUP BY
        source_id)
UPDATE
    sources s
SET
    novels_count = GREATEST (0, s.novels_count - ps.novels_delta)
FROM
    per_source ps
WHERE
    s.id = ps.source_id;
    WITH pairs AS (
        SELECT DISTINCT
            source_id,
            novel_id
        FROM
            new_rows
        WHERE
            source_id IS NOT NULL
),
new_pairs AS (
    SELECT
        p.source_id
    FROM
        pairs p
    WHERE
        NOT EXISTS (
            SELECT
                1
            FROM
                chapters c
            WHERE
                c.novel_id = p.novel_id
                AND c.source_id = p.source_id
                AND NOT EXISTS (
                    SELECT
                        1
                    FROM
                        new_rows nr
                    WHERE
                        nr.id = c.id))
                AND NOT EXISTS (
                    SELECT
                        1
                    FROM
                        old_rows orr
                    WHERE
                        orr.novel_id = p.novel_id
                        AND orr.source_id = p.source_id)
),
per_source AS (
    SELECT
        source_id,
        COUNT(*) AS novels_delta
    FROM
        new_pairs
    GROUP BY
        source_id)
UPDATE
    sources s
SET
    novels_count = s.novels_count + ps.novels_delta
FROM
    per_source ps
WHERE
    s.id = ps.source_id;
    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE OR REPLACE TRIGGER trg_source_stats_insert
    AFTER INSERT ON chapters REFERENCING NEW TABLE AS new_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION update_source_chapter_stats_insert ();

CREATE OR REPLACE TRIGGER trg_source_stats_delete
    AFTER DELETE ON chapters REFERENCING OLD TABLE AS old_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION update_source_chapter_stats_delete ();

CREATE OR REPLACE TRIGGER trg_source_stats_update
    AFTER UPDATE ON chapters REFERENCING OLD TABLE AS old_rows NEW TABLE AS new_rows
    FOR EACH STATEMENT
    EXECUTE FUNCTION update_source_chapter_stats_update ();

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

