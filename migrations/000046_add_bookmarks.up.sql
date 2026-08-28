ALTER TABLE users
    ADD COLUMN bookmarks jsonb NOT NULL DEFAULT '{}';

