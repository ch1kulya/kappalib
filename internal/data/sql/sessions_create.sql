INSERT INTO sessions (user_id, token_hash)
VALUES ($1, $2)
RETURNING id, created_at;
