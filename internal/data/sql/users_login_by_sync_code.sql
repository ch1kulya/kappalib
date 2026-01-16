SELECT id, cookies
FROM users
WHERE sync_code_hash = $1 AND sync_code_expires_at > now()
