UPDATE
    comment_answers
SET
    content_html = $1,
    status = 'pending',
    edited_at = now(),
    telegram_message_id = NULL
WHERE
    id = $2
    AND user_id = $3
RETURNING
    id,
    comment_id,
    user_id,
    content_html,
    status,
    edited_at,
    created_at;

