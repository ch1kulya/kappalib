SELECT
    cg.novel_id,
    n.title,
    n.cover_url,
    cg.chapter_min,
    cg.chapter_max,
    cg.chapter_count,
    cg.updated_at
FROM chapter_groups cg
JOIN novels n ON cg.novel_id = n.id
WHERE n.age_rating IS DISTINCT FROM '19+'
ORDER BY cg.updated_at DESC
LIMIT $1
