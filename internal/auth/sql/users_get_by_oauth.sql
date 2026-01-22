SELECT id FROM users
WHERE oauth_provider = $1 AND oauth_id = $2;
