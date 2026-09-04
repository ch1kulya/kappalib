SELECT
    id,
    text,
    url
FROM
    global_announcements
WHERE
    is_active = TRUE
ORDER BY
    random()
LIMIT 1;

