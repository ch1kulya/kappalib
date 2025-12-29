SELECT id, chapter_id, user_id, content_html, status, telegram_message_id, created_at
FROM comments WHERE id = $1;
