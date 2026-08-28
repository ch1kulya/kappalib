ALTER TABLE sources
    ADD COLUMN novels_count integer NOT NULL DEFAULT 0;

ALTER TABLE sources
    ADD COLUMN chapters_count integer NOT NULL DEFAULT 0;

ALTER TABLE sources
    ADD COLUMN views_count bigint NOT NULL DEFAULT 0;

CREATE INDEX idx_sources_novels_count ON sources (novels_count DESC);

CREATE INDEX idx_sources_chapters_count ON sources (chapters_count DESC);

CREATE INDEX idx_sources_views_count ON sources (views_count DESC);

CREATE OR REPLACE FUNCTION update_source_chapter_stats ()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.source_id IS NOT NULL THEN
            UPDATE
                sources
            SET
                chapters_count = chapters_count + 1
            WHERE
                id = NEW.source_id;
            IF NOT EXISTS (
                SELECT
                    1
                FROM
                    chapters
                WHERE
                    novel_id = NEW.novel_id
                    AND source_id = NEW.source_id
                    AND id != NEW.id) THEN
            UPDATE
                sources
            SET
                novels_count = novels_count + 1
            WHERE
                id = NEW.source_id;
        END IF;
    END IF;
    RETURN NEW;
ELSIF TG_OP = 'DELETE' THEN
    IF OLD.source_id IS NOT NULL THEN
        UPDATE
            sources
        SET
            chapters_count = chapters_count - 1
        WHERE
            id = OLD.source_id;
        IF NOT EXISTS (
            SELECT
                1
            FROM
                chapters
            WHERE
                novel_id = OLD.novel_id
                AND source_id = OLD.source_id) THEN
        UPDATE
            sources
        SET
            novels_count = novels_count - 1
        WHERE
            id = OLD.source_id;
    END IF;
END IF;
    RETURN OLD;
ELSIF TG_OP = 'UPDATE' THEN
    IF OLD.source_id IS DISTINCT FROM NEW.source_id THEN
        IF OLD.source_id IS NOT NULL THEN
            UPDATE
                sources
            SET
                chapters_count = chapters_count - 1
            WHERE
                id = OLD.source_id;
            IF NOT EXISTS (
                SELECT
                    1
                FROM
                    chapters
                WHERE
                    novel_id = OLD.novel_id
                    AND source_id = OLD.source_id
                    AND id != OLD.id) THEN
            UPDATE
                sources
            SET
                novels_count = novels_count - 1
            WHERE
                id = OLD.source_id;
        END IF;
    END IF;
    IF NEW.source_id IS NOT NULL THEN
        UPDATE
            sources
        SET
            chapters_count = chapters_count + 1
        WHERE
            id = NEW.source_id;
        IF NOT EXISTS (
            SELECT
                1
            FROM
                chapters
            WHERE
                novel_id = NEW.novel_id
                AND source_id = NEW.source_id
                AND id != NEW.id) THEN
        UPDATE
            sources
        SET
            novels_count = novels_count + 1
        WHERE
            id = NEW.source_id;
    END IF;
END IF;
END IF;
    RETURN NEW;
END IF;
    RETURN NULL;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER trg_update_source_chapter_stats
    AFTER INSERT OR UPDATE OR DELETE ON chapters
    FOR EACH ROW
    EXECUTE FUNCTION update_source_chapter_stats ();

CREATE OR REPLACE FUNCTION update_source_views_count ()
    RETURNS TRIGGER
    AS $$
BEGIN
    IF NEW.views_count > OLD.views_count THEN
        UPDATE
            sources
        SET
            views_count = views_count + (NEW.views_count - OLD.views_count)
        WHERE
            id IN ( SELECT DISTINCT
                    source_id
                FROM
                    chapters
                WHERE
                    novel_id = NEW.id
                    AND source_id IS NOT NULL);
    END IF;
    RETURN NEW;
END;
$$
LANGUAGE plpgsql;

CREATE TRIGGER trg_update_source_views
    AFTER UPDATE OF views_count ON novels
    FOR EACH ROW
    EXECUTE FUNCTION update_source_views_count ();

UPDATE
    sources s
SET
    chapters_count = COALESCE((
        SELECT
            COUNT(*)
        FROM chapters c
        WHERE
            c.source_id = s.id), 0), novels_count = COALESCE((
        SELECT
            COUNT(DISTINCT novel_id)
        FROM chapters c
        WHERE
            c.source_id = s.id), 0);

