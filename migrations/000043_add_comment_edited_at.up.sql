ALTER TABLE comments
    ADD COLUMN IF NOT EXISTS edited_at timestamptz DEFAULT NULL;

ALTER TABLE comment_answers
    ADD COLUMN IF NOT EXISTS edited_at timestamptz DEFAULT NULL;

