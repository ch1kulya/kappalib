SELECT
    c.id,
    c.novel_id,
    c.chapter_num,
    c.title,
    c.title_en,
    c.content,
    c.created_at,
    s.name,
    s.logo_url,
    s.label
FROM
    chapters AS c
    LEFT JOIN sources AS s ON c.source_id = s.id
WHERE
    c.novel_id = $1
    AND c.chapter_num > $2
ORDER BY
    c.chapter_num ASC
LIMIT $3;

