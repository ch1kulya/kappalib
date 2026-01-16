INSERT INTO users (display_name, avatar_seed, cookies)
VALUES ($1, $2, '{}')
RETURNING id, display_name, avatar_seed, created_at
