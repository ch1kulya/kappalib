SELECT
    COUNT(*)
FROM
    comments c
WHERE
    c.chapter_id = $1
    AND c.status != 'deleted'
    AND (c.status = 'approved'
        OR (c.status IN ('pending', 'rejected')
            AND c.user_id = $2));

