SELECT
    c.id,
    c.novel_id,
    c.chapter_num,
    c.title,
    c.title_en,
    c.content,
    c.created_at,
    s.name,
    s.logo_url
FROM chapters c
LEFT JOIN sources s ON c.source_id = s.id
WHERE c.id = $1;
