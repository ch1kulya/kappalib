SELECT
    n.id,
    COALESCE(n.title, ''),
    ch.id,
    COALESCE(ch.chapter_num, 0),
    COALESCE(ch.title, ''),
    COALESCE(u.display_name, ''),
    COALESCE(c.content_html, '')
FROM comments c
JOIN users u ON c.user_id = u.id
JOIN chapters ch ON c.chapter_id = ch.id
JOIN novels n ON ch.novel_id = n.id
WHERE c.id = $1;
