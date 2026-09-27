ALTER TABLE users
    DROP COLUMN IF EXISTS profile_comments_last_seen;

DELETE FROM comments
WHERE profile_id IS NOT NULL;

DROP INDEX IF EXISTS idx_comments_profile_status;

ALTER TABLE comments
    DROP CONSTRAINT IF EXISTS comments_target_check,
    DROP COLUMN IF EXISTS approved_at,
    DROP COLUMN IF EXISTS profile_id,
    ALTER COLUMN chapter_id SET NOT NULL;

