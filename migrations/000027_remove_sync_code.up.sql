DROP INDEX IF EXISTS idx_users_sync_code_hash;
ALTER TABLE users DROP COLUMN IF EXISTS sync_code_hash;
ALTER TABLE users DROP COLUMN IF EXISTS sync_code_expires_at;
