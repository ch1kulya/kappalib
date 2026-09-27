DROP INDEX IF EXISTS idx_comment_answers_notify;

CREATE INDEX IF NOT EXISTS idx_comment_answers_notify ON comment_answers (comment_id, created_at DESC)
WHERE
    status = 'approved';

ALTER TABLE comment_answers
    DROP COLUMN IF EXISTS approved_at;

