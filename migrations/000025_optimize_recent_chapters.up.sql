CREATE INDEX idx_chapters_created_at_novel ON chapters (created_at DESC, novel_id) INCLUDE (chapter_num);

CREATE INDEX idx_novels_age_rating ON novels (id) INCLUDE (title, cover_url, age_rating);

