ALTER TABLE sources ADD COLUMN views_count BIGINT NOT NULL DEFAULT 0;

CREATE INDEX idx_sources_views_count ON sources (views_count DESC);

CREATE OR REPLACE FUNCTION update_source_views_count()
RETURNS TRIGGER AS $$
BEGIN
    IF NEW.views_count > OLD.views_count THEN
        UPDATE sources
        SET views_count = views_count + (NEW.views_count - OLD.views_count)
        WHERE id IN (
            SELECT DISTINCT source_id
            FROM chapters
            WHERE novel_id = NEW.id AND source_id IS NOT NULL
        );
    END IF;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER trg_update_source_views
AFTER UPDATE OF views_count ON novels
FOR EACH ROW EXECUTE FUNCTION update_source_views_count();
