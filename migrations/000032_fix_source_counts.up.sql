ALTER TABLE sources ADD COLUMN IF NOT EXISTS characters_count BIGINT NOT NULL DEFAULT 0;

CREATE INDEX IF NOT EXISTS idx_sources_characters_count ON sources (characters_count DESC);

WITH stats AS (
    SELECT
        source_id,
        COUNT(*) AS chapters,
        COUNT(DISTINCT novel_id) AS novels,
        COALESCE(SUM(LENGTH(content)), 0) AS characters
    FROM chapters
    WHERE source_id IS NOT NULL
    GROUP BY source_id
)
UPDATE sources s SET
    chapters_count = COALESCE(stats.chapters, 0),
    novels_count = COALESCE(stats.novels, 0),
    characters_count = COALESCE(stats.characters, 0)
FROM stats
WHERE s.id = stats.source_id;

UPDATE sources SET chapters_count = 0, novels_count = 0, characters_count = 0
WHERE id NOT IN (SELECT DISTINCT source_id FROM chapters WHERE source_id IS NOT NULL);

CREATE OR REPLACE FUNCTION update_source_chapter_stats()
RETURNS TRIGGER AS $$
BEGIN
    IF TG_OP = 'INSERT' THEN
        IF NEW.source_id IS NOT NULL THEN
            UPDATE sources SET
                chapters_count = chapters_count + 1,
                characters_count = characters_count + LENGTH(NEW.content)
            WHERE id = NEW.source_id;

            IF NOT EXISTS (
                SELECT 1 FROM chapters
                WHERE novel_id = NEW.novel_id AND source_id = NEW.source_id AND id != NEW.id
            ) THEN
                UPDATE sources SET novels_count = novels_count + 1 WHERE id = NEW.source_id;
            END IF;
        END IF;
        RETURN NEW;

    ELSIF TG_OP = 'DELETE' THEN
        IF OLD.source_id IS NOT NULL THEN
            UPDATE sources SET
                chapters_count = GREATEST(0, chapters_count - 1),
                characters_count = GREATEST(0, characters_count - LENGTH(OLD.content))
            WHERE id = OLD.source_id;

            IF NOT EXISTS (
                SELECT 1 FROM chapters
                WHERE novel_id = OLD.novel_id AND source_id = OLD.source_id
            ) THEN
                UPDATE sources SET novels_count = GREATEST(0, novels_count - 1) WHERE id = OLD.source_id;
            END IF;
        END IF;
        RETURN OLD;

    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.source_id IS DISTINCT FROM NEW.source_id THEN
            IF OLD.source_id IS NOT NULL THEN
                UPDATE sources SET
                    chapters_count = GREATEST(0, chapters_count - 1),
                    characters_count = GREATEST(0, characters_count - LENGTH(OLD.content))
                WHERE id = OLD.source_id;
                IF NOT EXISTS (
                    SELECT 1 FROM chapters
                    WHERE novel_id = OLD.novel_id AND source_id = OLD.source_id AND id != OLD.id
                ) THEN
                    UPDATE sources SET novels_count = GREATEST(0, novels_count - 1) WHERE id = OLD.source_id;
                END IF;
            END IF;

            IF NEW.source_id IS NOT NULL THEN
                UPDATE sources SET
                    chapters_count = chapters_count + 1,
                    characters_count = characters_count + LENGTH(NEW.content)
                WHERE id = NEW.source_id;
                IF NOT EXISTS (
                    SELECT 1 FROM chapters
                    WHERE novel_id = NEW.novel_id AND source_id = NEW.source_id AND id != NEW.id
                ) THEN
                    UPDATE sources SET novels_count = novels_count + 1 WHERE id = NEW.source_id;
                END IF;
            END IF;
        ELSIF OLD.content IS DISTINCT FROM NEW.content AND NEW.source_id IS NOT NULL THEN
            UPDATE sources SET
                characters_count = GREATEST(0, characters_count - LENGTH(OLD.content)) + LENGTH(NEW.content)
            WHERE id = NEW.source_id;
        END IF;
        RETURN NEW;
    END IF;

    RETURN NULL;
END;
$$ LANGUAGE plpgsql;
