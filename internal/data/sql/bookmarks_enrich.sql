SELECT
    ch.id,
    n.id,
    n.title,
    ch.chapter_num
FROM
    chapters ch
    JOIN novels n ON ch.novel_id = n.id
WHERE
    ch.id = ANY ($1);

