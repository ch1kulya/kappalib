INSERT INTO comment_answers (comment_id, user_id, content_html, status)
VALUES ($1, $2, $3, 'pending')
RETURNING id, comment_id, user_id, content_html, status, created_at;
