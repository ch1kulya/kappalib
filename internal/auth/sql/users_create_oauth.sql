INSERT INTO users (display_name, avatar_seed, oauth_provider, oauth_id)
VALUES ($1, $2, $3, $4)
RETURNING id;
