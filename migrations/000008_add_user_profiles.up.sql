CREATE TABLE IF NOT EXISTS users (
    id varchar(20) PRIMARY KEY DEFAULT generate_short_id ('usr_'),
    secret_token varchar(64) NOT NULL,
    display_name varchar(100) NOT NULL,
    avatar_seed varchar(50) NOT NULL,
    cookies jsonb NOT NULL DEFAULT '{}',
    sync_code varchar(8) UNIQUE,
    sync_code_expires_at timestamptz,
    created_at timestamptz DEFAULT now(),
    last_active_at timestamptz DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_profiles_sync_code ON users (sync_code)
WHERE
    sync_code IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_profiles_last_active ON users (last_active_at);

