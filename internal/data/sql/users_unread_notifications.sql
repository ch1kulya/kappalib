SELECT
    ((
            SELECT
                COUNT(*)
            FROM
                comment_answers ca
                JOIN comments c ON ca.comment_id = c.id
            WHERE
                ca.user_id != u.id
                AND c.status = 'approved'
                AND ca.status = 'approved'
                AND ca.approved_at > COALESCE(u.notifications_last_seen, now() - INTERVAL '1 year')
                AND (c.user_id = u.id
                    OR EXISTS (
                        SELECT
                            1
                        FROM
                            comment_answers my_ca
                        WHERE
                            my_ca.comment_id = ca.comment_id
                            AND my_ca.user_id = u.id
                            AND my_ca.status != 'deleted'))) + (
                    SELECT
                        COUNT(*)
                    FROM
                        comments pc
                    WHERE
                        pc.profile_id = u.id
                        AND pc.user_id != u.id
                        AND pc.status = 'approved'
                        AND pc.approved_at > u.profile_comments_last_seen))::int
        FROM
            users u
        WHERE
            u.id = $1;

