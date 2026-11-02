ALTER TABLE comment_answers
    ALTER COLUMN created_at DROP NOT NULL;

ALTER TABLE comments
    ALTER COLUMN created_at DROP NOT NULL;

ALTER TABLE chapters
    ALTER COLUMN created_at DROP NOT NULL;

ALTER TABLE novels
    ALTER COLUMN created_at DROP NOT NULL,
    ALTER COLUMN chapters_count DROP NOT NULL;

ALTER TABLE users
    ALTER COLUMN created_at DROP NOT NULL,
    ALTER COLUMN avatar_updated_at DROP NOT NULL,
    ALTER COLUMN has_custom_avatar DROP NOT NULL;

