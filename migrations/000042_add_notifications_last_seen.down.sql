DROP INDEX IF EXISTS idx_comment_answers_notify;
ALTER TABLE users DROP COLUMN IF EXISTS notifications_last_seen;
