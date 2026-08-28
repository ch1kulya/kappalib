UPDATE
    users
SET
    has_custom_avatar = TRUE,
    avatar_updated_at = now(),
    last_active_at = now()
WHERE
    id = $1;

