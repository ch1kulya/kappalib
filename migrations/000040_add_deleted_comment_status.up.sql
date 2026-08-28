ALTER TABLE comments
    DROP CONSTRAINT comments_status_check;

ALTER TABLE comments
    ADD CONSTRAINT comments_status_check CHECK (status IN ('pending', 'approved', 'rejected', 'deleted'));

