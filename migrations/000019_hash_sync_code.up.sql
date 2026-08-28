ALTER TABLE users
    ADD COLUMN IF NOT EXISTS sync_code_hash varchar(64);

UPDATE
    users
SET
    sync_code_hash = encode(sha256 (sync_code::bytea), 'hex')
WHERE
    sync_code IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_users_sync_code_hash ON users (sync_code_hash)
WHERE
    sync_code_hash IS NOT NULL;

DROP INDEX IF EXISTS idx_profiles_sync_code;

ALTER TABLE users
    DROP COLUMN IF EXISTS sync_code;

