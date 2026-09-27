ALTER TABLE comment_answers
    ADD COLUMN IF NOT EXISTS approved_at timestamptz;

UPDATE
    comment_answers
SET
    approved_at = created_at
WHERE
    status = 'approved'
    AND approved_at IS NULL;

DROP INDEX IF EXISTS idx_comment_answers_notify;

CREATE INDEX IF NOT EXISTS idx_comment_answers_notify ON comment_answers (comment_id, approved_at DESC)
WHERE
    status = 'approved';

