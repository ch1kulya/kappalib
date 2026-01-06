CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(20) PRIMARY KEY DEFAULT generate_short_id('ses_'),
    user_id VARCHAR(20) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL UNIQUE,
    device_info TEXT,
    created_at TIMESTAMPTZ DEFAULT now(),
    last_used_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_sessions_user_id ON sessions(user_id);
CREATE INDEX IF NOT EXISTS idx_sessions_token_hash ON sessions(token_hash);
CREATE INDEX IF NOT EXISTS idx_sessions_last_used ON sessions(user_id, last_used_at DESC);

INSERT INTO sessions (user_id, token_hash, created_at, last_used_at)
SELECT id, token_hash, created_at, last_active_at
FROM users
WHERE token_hash IS NOT NULL;

DROP INDEX IF EXISTS idx_users_token_hash;
ALTER TABLE users DROP COLUMN IF EXISTS token_hash;
