CREATE TABLE IF NOT EXISTS comments (
    id varchar(20) PRIMARY KEY DEFAULT generate_short_id ('cmt_'),
    chapter_id varchar(20) NOT NULL REFERENCES chapters (id) ON DELETE CASCADE,
    user_id varchar(20) NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    content_html text NOT NULL,
    status varchar(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected')),
    telegram_message_id bigint,
    created_at timestamptz DEFAULT now()
);

CREATE INDEX IF NOT EXISTS idx_comments_chapter_id ON comments (chapter_id);

CREATE INDEX IF NOT EXISTS idx_comments_user_id ON comments (user_id);

CREATE INDEX IF NOT EXISTS idx_comments_status ON comments (status);

CREATE INDEX IF NOT EXISTS idx_comments_chapter_status ON comments (chapter_id, status, created_at DESC);

