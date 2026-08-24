SELECT
    id,
    title,
    author,
    year_start,
    status,
    description,
    cover_url,
    created_at
FROM novels
ORDER BY created_at DESC
LIMIT NULLIF($1, 0);
