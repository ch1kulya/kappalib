CREATE TABLE comment_votes (
    id varchar(20) PRIMARY KEY DEFAULT generate_short_id ('cvt_'),
    comment_id varchar(20) NOT NULL REFERENCES comments (id) ON DELETE CASCADE,
    user_id varchar(20) NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    value smallint NOT NULL CHECK (value IN (-1, 1)),
    created_at timestamptz DEFAULT now(),
    UNIQUE (comment_id, user_id)
);

CREATE INDEX idx_comment_votes_comment ON comment_votes (comment_id);

CREATE INDEX idx_comment_votes_user ON comment_votes (user_id);

