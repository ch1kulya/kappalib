UPDATE users
SET display_name = $1, last_active_at = now()
WHERE id = $2;
