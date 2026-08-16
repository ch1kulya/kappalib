SELECT
    n.id,
    COALESCE(n.title, ''),
    ch.id,
    COALESCE(ch.chapter_num, 0),
    COALESCE(ch.title, '')
FROM chapters ch
JOIN novels n ON ch.novel_id = n.id
WHERE ch.id = $1;
