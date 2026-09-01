SELECT
    c.id,
    c.chapter_id,
    COALESCE(c.content_html, ''),
    COALESCE(u.display_name, '')
FROM
    comments c
    JOIN users u ON c.user_id = u.id
WHERE
    c.status = 'pending'
    AND c.telegram_message_id IS NULL
    AND c.created_at < NOW() - INTERVAL '2 minutes'
ORDER BY
    c.created_at
LIMIT 25;

