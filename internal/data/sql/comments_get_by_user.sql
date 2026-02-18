SELECT
    c.id, c.chapter_id, c.content_html, c.status, c.created_at,
    ch.chapter_num, ch.novel_id,
    n.title AS novel_title,
    COALESCE((SELECT SUM(value) FROM comment_votes WHERE comment_id = c.id), 0)::int AS score,
    COALESCE((SELECT value FROM comment_votes WHERE comment_id = c.id AND user_id = $1), 0)::int AS user_vote
FROM comments c
JOIN chapters ch ON c.chapter_id = ch.id
JOIN novels n ON ch.novel_id = n.id
WHERE c.user_id = $1
  AND c.status != 'deleted'
ORDER BY c.created_at DESC
LIMIT $2 OFFSET $3;
