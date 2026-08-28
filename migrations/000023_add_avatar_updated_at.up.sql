ALTER TABLE users
    ADD COLUMN IF NOT EXISTS avatar_updated_at timestamptz DEFAULT now();

