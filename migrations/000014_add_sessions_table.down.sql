ALTER TABLE users
    ADD COLUMN IF NOT EXISTS token_hash varchar(64);

UPDATE
    users u
SET
    token_hash = s.token_hash
FROM ( SELECT DISTINCT ON (user_id)
        user_id,
        token_hash
    FROM
        sessions
    ORDER BY
        user_id,
        last_used_at DESC) s
WHERE
    u.id = s.user_id;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_token_hash ON users (token_hash);

DROP INDEX IF EXISTS idx_sessions_user_id;

DROP INDEX IF EXISTS idx_sessions_token_hash;

DROP INDEX IF EXISTS idx_sessions_last_used;

DROP TABLE IF EXISTS sessions;

