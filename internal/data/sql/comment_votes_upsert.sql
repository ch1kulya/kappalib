INSERT INTO comment_votes (comment_id, user_id, value)
VALUES ($1, $2, $3)
ON CONFLICT (comment_id, user_id)
DO UPDATE SET value = $3
RETURNING id;
