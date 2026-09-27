SELECT
    user_id,
    status,
    COALESCE(profile_id, '')
FROM
    comments
WHERE
    id = $1;

