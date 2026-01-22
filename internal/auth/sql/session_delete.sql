DELETE FROM sessions
WHERE token_hash = $1;
