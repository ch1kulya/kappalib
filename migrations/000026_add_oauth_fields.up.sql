ALTER TABLE users
    ADD COLUMN IF NOT EXISTS email varchar(255);

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS oauth_provider varchar(50);

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS oauth_id varchar(255);

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_oauth ON users (oauth_provider, oauth_id)
WHERE
    oauth_provider IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS idx_users_email ON users (email)
WHERE
    email IS NOT NULL;

