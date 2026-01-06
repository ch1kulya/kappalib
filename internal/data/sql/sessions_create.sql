INSERT INTO sessions (user_id, token_hash, device_info)
VALUES ($1, $2, $3)
RETURNING id, created_at;
