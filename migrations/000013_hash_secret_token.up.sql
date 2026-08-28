ALTER TABLE users
    ADD COLUMN IF NOT EXISTS token_hash varchar(64);

UPDATE
    users
SET
    token_hash = encode(sha256 (secret_token::bytea), 'hex')
WHERE
    secret_token IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_token_hash ON users (token_hash);

ALTER TABLE users
    DROP COLUMN IF EXISTS secret_token;

