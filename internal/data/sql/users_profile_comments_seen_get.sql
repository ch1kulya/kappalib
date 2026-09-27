SELECT
    profile_comments_last_seen
FROM
    users
WHERE
    id = $1;

