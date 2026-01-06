DELETE FROM sessions
WHERE user_id = $1 AND id NOT IN (
    SELECT id FROM sessions
    WHERE user_id = $1
    ORDER BY last_used_at DESC
    LIMIT $2
);
