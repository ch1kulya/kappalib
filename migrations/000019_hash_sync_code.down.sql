ALTER TABLE users
    ADD COLUMN IF NOT EXISTS sync_code varchar(24);

DROP INDEX IF EXISTS idx_users_sync_code_hash;

ALTER TABLE users
    DROP COLUMN IF EXISTS sync_code_hash;

CREATE INDEX IF NOT EXISTS idx_profiles_sync_code ON users (sync_code)
WHERE
    sync_code IS NOT NULL;

