CREATE VIEW data_issues AS
SELECT 'chapter_short_content' AS issue, id, novel_id AS related_id, title, created_at
FROM chapters WHERE LENGTH(content) < 100
UNION ALL
SELECT 'chapter_empty_title', id, novel_id, title, created_at
FROM chapters WHERE TRIM(title) = ''
UNION ALL
SELECT 'novel_empty_title', id, id, title, created_at
FROM novels WHERE TRIM(title) = ''
UNION ALL
SELECT 'novel_empty_description', id, id, title, created_at
FROM novels WHERE TRIM(description) = ''
UNION ALL
SELECT 'novel_no_cover', id, id, title, created_at
FROM novels WHERE cover_url = '' OR cover_url IS NULL;

CREATE INDEX idx_chapters_short_content ON chapters (id) WHERE LENGTH(content) < 100;
CREATE INDEX idx_chapters_empty_title ON chapters (id) WHERE TRIM(title) = '';
CREATE INDEX idx_novels_empty_title ON novels (id) WHERE TRIM(title) = '';
CREATE INDEX idx_novels_empty_description ON novels (id) WHERE TRIM(description) = '';
CREATE INDEX idx_novels_no_cover ON novels (id) WHERE cover_url = '' OR cover_url IS NULL;
