SELECT
    COALESCE((
        SELECT
            SUM(cv.value)
        FROM comment_votes cv
        JOIN comments c ON cv.comment_id = c.id
        WHERE
            c.user_id = $1
            AND c.status = 'approved'
            AND cv.created_at < CURRENT_DATE - INTERVAL '29 days'), 0),
    COALESCE((
        SELECT
            COUNT(*)
        FROM comment_answers ca
        JOIN comments c ON ca.comment_id = c.id
        WHERE
            c.user_id = $1
            AND ca.user_id != $1
            AND ca.status = 'approved'
            AND c.status = 'approved'
            AND ca.approved_at < CURRENT_DATE - INTERVAL '29 days'), 0)
