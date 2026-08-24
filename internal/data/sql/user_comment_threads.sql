SELECT id, MAX(last_activity) AS last_activity FROM (
    SELECT id, created_at AS last_activity
    FROM comments
    WHERE user_id = $1 AND status != 'deleted'

    UNION ALL

    SELECT comment_id AS id, created_at AS last_activity
    FROM comment_answers
    WHERE user_id = $1 AND status != 'deleted'

    UNION ALL

    SELECT ca.comment_id AS id, ca.created_at AS last_activity
    FROM comment_answers ca
    JOIN comments c ON ca.comment_id = c.id
    WHERE ca.status = 'approved'
      AND ca.user_id != $1
      AND (
        c.user_id = $1
        OR EXISTS (
          SELECT 1
          FROM comment_answers my_ca
          WHERE my_ca.comment_id = ca.comment_id
            AND my_ca.user_id = $1
            AND my_ca.status != 'deleted'
        )
      )
) AS combined
GROUP BY id
ORDER BY last_activity DESC
LIMIT $2 OFFSET $3;
