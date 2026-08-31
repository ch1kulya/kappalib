ALTER TABLE users
    ADD COLUMN novel_list jsonb NOT NULL DEFAULT '{}';

