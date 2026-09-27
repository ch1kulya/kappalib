SELECT
    c.id,
    COALESCE(c.chapter_id, ''),
    COALESCE(c.profile_id, ''),
    c.user_id,
    c.content_html,
    c.status,
    c.edited_at,
    c.created_at,
    c.approved_at,
    u.display_name,
    u.avatar_seed,
    u.has_custom_avatar,
    u.avatar_updated_at,
    COALESCE((
        SELECT
            SUM(value)
        FROM comment_votes
        WHERE
            comment_id = c.id), 0)::int,
    COALESCE((
        SELECT
            value
        FROM comment_votes
        WHERE
            comment_id = c.id
            AND user_id = $2), 0)::int
FROM
    comments c
    JOIN users u ON c.user_id = u.id
WHERE
    c.chapter_id = $1
    AND c.status != 'deleted'
    AND (c.status = 'approved'
        OR (c.status IN ('pending', 'rejected')
            AND c.user_id = $2))
ORDER BY
    c.created_at DESC,
    c.id ASC
LIMIT $3 OFFSET $4;

