SELECT
    ca.id,
    ca.comment_id,
    ca.user_id,
    ca.content_html,
    ca.status,
    ca.edited_at,
    ca.created_at,
    ca.approved_at,
    u.display_name,
    u.avatar_seed,
    u.has_custom_avatar,
    u.avatar_updated_at
FROM
    comment_answers ca
    JOIN users u ON ca.user_id = u.id
WHERE
    ca.comment_id = ANY ($1)
    AND ca.status != 'deleted'
    AND (ca.status = 'approved'
        OR (ca.status IN ('pending', 'rejected')
            AND ca.user_id = $2))
ORDER BY
    ca.comment_id,
    ca.created_at ASC;

