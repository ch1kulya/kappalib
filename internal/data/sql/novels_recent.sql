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
WHERE
    age_rating IS DISTINCT FROM '19+'
    AND created_at > NOW() - INTERVAL '30 days'
ORDER BY created_at DESC
LIMIT $1;
