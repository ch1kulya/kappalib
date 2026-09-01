DROP TRIGGER IF EXISTS trg_update_source_chapter_stats_insert ON chapters;
DROP TRIGGER IF EXISTS trg_update_source_chapter_stats_delete ON chapters;
DROP TRIGGER IF EXISTS trg_update_source_chapter_stats_update ON chapters;
DROP TRIGGER IF EXISTS trg_update_source_chapter_stats ON chapters;

DROP INDEX IF EXISTS idx_sources_characters_count;

ALTER TABLE sources
    DROP COLUMN IF EXISTS characters_count;

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

DROP TRIGGER IF EXISTS trg_update_source_chapter_stats ON chapters;
CREATE TRIGGER trg_update_source_chapter_stats
    AFTER INSERT OR UPDATE OR DELETE ON chapters
    FOR EACH ROW
    EXECUTE FUNCTION update_source_chapter_stats ();