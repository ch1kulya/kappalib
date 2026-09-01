SELECT
    ca.id,
    ca.comment_id,
    COALESCE(ca.content_html, ''),
    COALESCE(u.display_name, '')
FROM
    comment_answers ca
    JOIN users u ON ca.user_id = u.id
WHERE
    ca.status = 'pending'
    AND ca.telegram_message_id IS NULL
    AND ca.created_at < NOW() - INTERVAL '2 minutes'
ORDER BY
    ca.created_at
LIMIT 25;

