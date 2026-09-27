ALTER TABLE comments
    ALTER COLUMN chapter_id DROP NOT NULL,
    ADD COLUMN IF NOT EXISTS profile_id varchar(20) REFERENCES users (id) ON DELETE CASCADE,
    ADD COLUMN IF NOT EXISTS approved_at timestamptz;

ALTER TABLE comments
    ADD CONSTRAINT comments_target_check CHECK ((chapter_id IS NULL) <> (profile_id IS NULL));

UPDATE
    comments
SET
    approved_at = created_at
WHERE
    status = 'approved'
    AND approved_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_comments_profile_status ON comments (profile_id, status, created_at DESC)
WHERE
    profile_id IS NOT NULL;

ALTER TABLE users
    ADD COLUMN IF NOT EXISTS profile_comments_last_seen timestamptz NOT NULL DEFAULT now();

