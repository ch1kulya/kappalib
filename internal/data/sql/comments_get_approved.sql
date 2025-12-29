SELECT
    c.id, c.chapter_id, c.user_id, c.content_html, c.status, c.created_at,
    u.display_name, u.avatar_seed
FROM comments c
JOIN users u ON c.user_id = u.id
WHERE c.chapter_id = $1 AND c.status = 'approved'
ORDER BY c.created_at DESC
LIMIT $2 OFFSET $3;
