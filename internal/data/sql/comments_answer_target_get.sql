SELECT
    status,
    profile_id IS NOT NULL
FROM
    comments
WHERE
    id = $1;

