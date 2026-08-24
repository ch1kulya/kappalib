SELECT
    cg.novel_id,
    n.title,
    n.cover_url,
    n.author,
    n.year_start,
    n.status,
    n.description,
    cg.chapter_min,
    cg.chapter_max,
    cg.chapter_count,
    cg.updated_at
FROM chapter_groups AS cg
INNER JOIN novels AS n ON cg.novel_id = n.id
ORDER BY cg.updated_at DESC
LIMIT NULLIF($1, 0);
