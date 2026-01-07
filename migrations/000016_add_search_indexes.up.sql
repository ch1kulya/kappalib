CREATE INDEX IF NOT EXISTS idx_novels_title_norm_btree ON novels (title_norm);
CREATE INDEX IF NOT EXISTS idx_novels_title_en_norm_btree ON novels (title_en_norm);
CREATE INDEX IF NOT EXISTS idx_novels_author_norm_btree ON novels (author_norm);
