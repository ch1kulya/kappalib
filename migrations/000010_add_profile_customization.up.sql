ALTER TABLE users
    ADD COLUMN IF NOT EXISTS has_custom_avatar boolean DEFAULT FALSE;

