SELECT
    c.novel_id,
    n.title,
    n.cover_url,
    MIN(c.chapter_num) as chapter_min,
    MAX(c.chapter_num) as chapter_max,
    COUNT(*) as chapter_count,
    MAX(c.created_at) as updated_at
FROM chapters c
JOIN novels n ON c.novel_id = n.id
WHERE n.age_rating IS DISTINCT FROM '19+'
  AND c.created_at > NOW() - INTERVAL '7 days'
GROUP BY c.novel_id, n.title, n.cover_url, date_trunc('hour', c.created_at)
ORDER BY updated_at DESC
LIMIT $1
