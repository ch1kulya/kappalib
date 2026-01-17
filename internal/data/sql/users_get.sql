SELECT id, display_name, avatar_seed, has_custom_avatar, avatar_updated_at, created_at
FROM users
WHERE id = $1
