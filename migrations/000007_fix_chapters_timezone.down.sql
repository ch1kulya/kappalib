ALTER TABLE novels
    ALTER COLUMN created_at TYPE timestamp
    USING created_at AT TIME ZONE 'Europe/Moscow';

ALTER TABLE novels
    ALTER COLUMN created_at SET DEFAULT CURRENT_TIMESTAMP;

ALTER TABLE chapters
    ALTER COLUMN created_at TYPE timestamp
    USING created_at AT TIME ZONE 'Europe/Moscow';

ALTER TABLE chapters
    ALTER COLUMN created_at SET DEFAULT CURRENT_TIMESTAMP;
