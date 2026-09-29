SELECT
    t.id,
    t.name
FROM
    tags AS t
WHERE
    EXISTS (
        SELECT
            1
        FROM
            novel_tags AS nt
        WHERE
            nt.tag_id = t.id)
ORDER BY
    t.name;

