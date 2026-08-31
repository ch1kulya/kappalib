SELECT
    n.id,
    n.title,
    n.author,
    n.cover_url
FROM
    novels n
WHERE
    n.id = ANY ($1);

