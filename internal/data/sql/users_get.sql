SELECT id, display_name, avatar_seed, has_custom_avatar, created_at
FROM users
WHERE id = $1
