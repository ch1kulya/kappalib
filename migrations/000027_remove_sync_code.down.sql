ALTER TABLE users ADD COLUMN IF NOT EXISTS sync_code_hash VARCHAR(64);
ALTER TABLE users ADD COLUMN IF NOT EXISTS sync_code_expires_at TIMESTAMPTZ;
CREATE INDEX IF NOT EXISTS idx_users_sync_code_hash ON users(sync_code_hash) WHERE sync_code_hash IS NOT NULL;
