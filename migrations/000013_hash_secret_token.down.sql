ALTER TABLE users
    ADD COLUMN IF NOT EXISTS secret_token varchar(64);

DROP INDEX IF EXISTS idx_users_token_hash;

ALTER TABLE users
    DROP COLUMN IF EXISTS token_hash;

