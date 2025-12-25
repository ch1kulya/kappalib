CREATE TABLE IF NOT EXISTS users (
    id VARCHAR(20) PRIMARY KEY DEFAULT generate_short_id('usr_'),
    secret_token VARCHAR(64) NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    avatar_seed VARCHAR(50) NOT NULL,
    cookies JSONB NOT NULL DEFAULT '{}',
    sync_code VARCHAR(8) UNIQUE,
    sync_code_expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT now(),
    last_active_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_profiles_sync_code ON users(sync_code) WHERE sync_code IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_profiles_last_active ON users(last_active_at);
