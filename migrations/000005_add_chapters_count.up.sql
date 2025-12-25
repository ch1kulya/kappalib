ALTER TABLE novels ADD COLUMN IF NOT EXISTS chapters_count INTEGER DEFAULT 0;

WITH counts AS (
    SELECT novel_id, COUNT(*) as cnt FROM chapters GROUP BY novel_id
)
UPDATE novels n
SET chapters_count = c.cnt
FROM counts c
WHERE n.id = c.novel_id;

CREATE INDEX IF NOT EXISTS idx_novels_chapters_count ON novels (chapters_count DESC);

CREATE OR REPLACE FUNCTION update_novel_chapter_count() RETURNS TRIGGER AS $$
BEGIN
    IF (TG_OP = 'INSERT') THEN
        UPDATE novels SET chapters_count = chapters_count + 1 WHERE id = NEW.novel_id;
    ELSIF (TG_OP = 'DELETE') THEN
        UPDATE novels SET chapters_count = chapters_count - 1 WHERE id = OLD.novel_id;
    END IF;
    RETURN NULL;
END;
$$ LANGUAGE plpgsql;

DROP TRIGGER IF EXISTS trg_update_novel_chapter_count ON chapters;

CREATE TRIGGER trg_update_novel_chapter_count
AFTER INSERT OR DELETE ON chapters
FOR EACH ROW EXECUTE FUNCTION update_novel_chapter_count();
