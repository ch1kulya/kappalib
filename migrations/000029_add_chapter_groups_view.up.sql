CREATE MATERIALIZED VIEW chapter_groups AS
SELECT
    novel_id,
    MIN(chapter_num) as chapter_min,
    MAX(chapter_num) as chapter_max,
    COUNT(*) as chapter_count,
    MAX(created_at) as updated_at
FROM (
    SELECT
        novel_id,
        chapter_num,
        created_at,
        SUM(new_group) OVER (PARTITION BY novel_id ORDER BY created_at) as grp
    FROM (
        SELECT
            novel_id,
            chapter_num,
            created_at,
            CASE
                WHEN created_at - LAG(created_at) OVER (PARTITION BY novel_id ORDER BY created_at) > INTERVAL '1 hour'
                THEN 1
                ELSE 0
            END as new_group
        FROM chapters
    ) with_flags
) with_groups
GROUP BY novel_id, grp;

CREATE UNIQUE INDEX idx_chapter_groups_pk ON chapter_groups(novel_id, updated_at);
CREATE INDEX idx_chapter_groups_updated ON chapter_groups(updated_at DESC);

CREATE OR REPLACE FUNCTION refresh_chapter_groups()
RETURNS TRIGGER AS $$
BEGIN
    REFRESH MATERIALIZED VIEW CONCURRENTLY chapter_groups;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_refresh_chapter_groups
AFTER INSERT OR DELETE ON chapters
FOR EACH STATEMENT EXECUTE FUNCTION refresh_chapter_groups();
