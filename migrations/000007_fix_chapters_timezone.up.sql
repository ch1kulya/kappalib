ALTER TABLE novels
    ALTER COLUMN created_at TYPE timestamptz
    USING created_at AT TIME ZONE 'Europe/Moscow';

ALTER TABLE novels
    ALTER COLUMN created_at SET DEFAULT now();

ALTER TABLE chapters
    ALTER COLUMN created_at TYPE timestamptz
    USING created_at AT TIME ZONE 'Europe/Moscow';

ALTER TABLE chapters
    ALTER COLUMN created_at SET DEFAULT now();
