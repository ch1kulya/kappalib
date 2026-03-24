SELECT id, MAX(created_at) AS last_activity FROM (
    SELECT id, created_at FROM comments WHERE user_id = $1 AND status != 'deleted'
    UNION ALL
    SELECT comment_id AS id, created_at FROM comment_answers WHERE user_id = $1 AND status != 'deleted'
) AS combined
GROUP BY id
ORDER BY last_activity DESC
LIMIT $2 OFFSET $3;
