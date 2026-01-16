UPDATE users
SET has_custom_avatar = true, last_active_at = now()
WHERE id = $1
