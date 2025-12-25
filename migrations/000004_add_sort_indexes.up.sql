CREATE INDEX IF NOT EXISTS idx_novels_year_title ON novels (year_start, title);
CREATE INDEX IF NOT EXISTS idx_novels_created_at_sort ON novels (created_at DESC);
CREATE INDEX IF NOT EXISTS idx_novels_title_cyrillic ON novels ( (regexp_replace(lower(title), '[^а-яё]', '', 'g')) );
