ALTER TABLE users
    ADD COLUMN notifications_last_seen timestamptz;

UPDATE
    users
SET
    notifications_last_seen = now();

ALTER TABLE users
    ALTER COLUMN notifications_last_seen SET DEFAULT now();

CREATE INDEX IF NOT EXISTS idx_comment_answers_notify ON comment_answers (comment_id, created_at DESC)
WHERE
    status = 'approved';

