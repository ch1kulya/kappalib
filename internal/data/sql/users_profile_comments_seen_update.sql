UPDATE
    users
SET
    profile_comments_last_seen = now()
WHERE
    id = $1;

