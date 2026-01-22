INSERT INTO sessions (user_id, token_hash)
VALUES ($1, $2)
ON CONFLICT (token_hash) DO UPDATE SET last_used_at = now()
RETURNING id;
