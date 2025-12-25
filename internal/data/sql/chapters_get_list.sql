SELECT id, chapter_num, title, title_en FROM chapters
WHERE novel_id = $1
ORDER BY chapter_num ASC;
