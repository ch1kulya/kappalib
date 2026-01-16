UPDATE users
SET sync_code_hash = $1, sync_code_expires_at = $2, last_active_at = now()
WHERE id = $3
