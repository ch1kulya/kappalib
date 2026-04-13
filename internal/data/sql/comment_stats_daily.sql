SELECT d.day::text,
       COALESCE(dv.val, 0),
       COALESCE(dr.cnt, 0)
FROM generate_series(
        CURRENT_DATE - INTERVAL '29 days',
        CURRENT_DATE,
        INTERVAL '1 day'
) AS d(day)
LEFT JOIN (
        SELECT cv.created_at::date AS day, COALESCE(SUM(cv.value), 0) AS val
        FROM comment_votes cv JOIN comments c ON cv.comment_id = c.id
        WHERE c.user_id = $1 AND c.status != 'deleted'
        AND cv.created_at >= CURRENT_DATE - INTERVAL '29 days'
        GROUP BY 1
) dv ON dv.day = d.day
LEFT JOIN (
        SELECT ca.created_at::date AS day, COUNT(*) AS cnt
        FROM comment_answers ca JOIN comments c ON ca.comment_id = c.id
        WHERE c.user_id = $1 AND ca.user_id != $1 AND ca.status != 'deleted' AND c.status != 'deleted'
        AND ca.created_at >= CURRENT_DATE - INTERVAL '29 days'
        GROUP BY 1
) dr ON dr.day = d.day
ORDER BY d.day
