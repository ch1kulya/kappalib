INSERT INTO comments (profile_id, user_id, content_html, status)
    VALUES ($1, $2, $3, 'pending')
RETURNING
    id, user_id, content_html, status, edited_at, created_at, profile_id;

