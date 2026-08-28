ALTER TABLE sessions
    ADD COLUMN IF NOT EXISTS device_info text;

