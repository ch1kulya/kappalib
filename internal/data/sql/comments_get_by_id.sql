SELECT
    id,
    chapter_id,
    user_id,
    content_html,
    status,
    edited_at,
    telegram_message_id,
    created_at
FROM comments
WHERE id = $1;
