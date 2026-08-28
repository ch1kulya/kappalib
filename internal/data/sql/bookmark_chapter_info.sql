SELECT
    n.id,
    n.title,
    ch.chapter_num,
    ch.title
FROM
    chapters ch
    JOIN novels n ON ch.novel_id = n.id
WHERE
    ch.id = $1;
