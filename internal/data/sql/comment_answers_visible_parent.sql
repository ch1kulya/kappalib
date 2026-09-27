SELECT
    comment_id
FROM
    comment_answers
WHERE
    id = $1
    AND status != 'deleted'
    AND (status = 'approved'
        OR (status IN ('pending', 'rejected')
            AND user_id = $2));

