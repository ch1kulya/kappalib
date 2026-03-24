CREATE TABLE IF NOT EXISTS comment_answers (
    id VARCHAR(20) PRIMARY KEY DEFAULT generate_short_id('can_'),
    comment_id VARCHAR(20) NOT NULL REFERENCES comments(id) ON DELETE CASCADE,
    user_id VARCHAR(20) NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content_html TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'approved', 'rejected', 'deleted')),
    telegram_message_id BIGINT,
    created_at TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_comment_answers_comment_id ON comment_answers(comment_id);
CREATE INDEX idx_comment_answers_user_id ON comment_answers(user_id);
CREATE INDEX idx_comment_answers_status ON comment_answers(status);
CREATE INDEX idx_comment_answers_created ON comment_answers(comment_id, status, created_at DESC);
