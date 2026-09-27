WITH target AS (
    SELECT
        id,
        created_at
    FROM
        comments
    WHERE
        id = $3
        AND chapter_id = $1
        AND status != 'deleted'
        AND (status = 'approved'
            OR (status IN ('pending', 'rejected')
                AND user_id = $2))
)
SELECT
    EXISTS (
        SELECT
            1
        FROM
            target),
    (
        SELECT
            COUNT(*)
        FROM
            comments c,
            target t
        WHERE
            c.chapter_id = $1
            AND c.status != 'deleted'
            AND (c.status = 'approved'
                OR (c.status IN ('pending', 'rejected')
                    AND c.user_id = $2))
            AND (c.created_at > t.created_at
                OR (c.created_at = t.created_at
                    AND c.id < t.id)));

