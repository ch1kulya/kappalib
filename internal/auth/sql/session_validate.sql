UPDATE sessions SET last_used_at = now()
WHERE token_hash = $1
RETURNING user_id;
