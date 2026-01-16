UPDATE users
SET sync_code_hash = NULL, sync_code_expires_at = NULL, last_active_at = now()
WHERE id = $1
