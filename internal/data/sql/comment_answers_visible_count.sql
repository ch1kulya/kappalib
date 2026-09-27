SELECT
    COUNT(*)
FROM
    comment_answers ca
    JOIN comments c ON ca.comment_id = c.id
WHERE
    c.chapter_id = $1
    AND c.status != 'deleted'
    AND (c.status = 'approved'
        OR (c.status IN ('pending', 'rejected')
            AND c.user_id = $2))
    AND ca.status != 'deleted'
    AND (ca.status = 'approved'
        OR (ca.status IN ('pending', 'rejected')
            AND ca.user_id = $2));

