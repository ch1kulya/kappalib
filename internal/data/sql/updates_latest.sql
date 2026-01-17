SELECT
    c.novel_id,
    n.title,
    n.cover_url,
    c.chapter_min,
    c.chapter_max,
    c.chapter_count,
    c.updated_at
FROM (
    SELECT
        novel_id,
        MIN(chapter_num) as chapter_min,
        MAX(chapter_num) as chapter_max,
        COUNT(*) as chapter_count,
        MAX(created_at) as updated_at
    FROM chapters
    WHERE created_at > NOW() - INTERVAL '7 days'
    GROUP BY novel_id
) c
JOIN novels n ON c.novel_id = n.id
WHERE n.age_rating IS DISTINCT FROM '19+'
ORDER BY c.updated_at DESC
LIMIT $1
