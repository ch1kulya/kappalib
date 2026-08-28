ALTER TABLE novels
    ADD COLUMN views_count bigint NOT NULL DEFAULT 0;

CREATE INDEX idx_novels_views_count ON novels (views_count DESC);

