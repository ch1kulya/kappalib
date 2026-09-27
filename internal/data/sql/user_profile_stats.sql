SELECT
    COALESCE((
        SELECT
            SUM(seconds_spent)
        FROM user_daily_time
        WHERE
            user_id = $1), 0)::int,
    COALESCE((
        SELECT
            MAX(streak)
        FROM (
            SELECT
                COUNT(*) AS streak FROM (
                SELECT
                    udt.date - (ROW_NUMBER() OVER (ORDER BY udt.date))::int AS grp FROM user_daily_time udt
            WHERE
                udt.user_id = $1
                AND udt.seconds_spent > 0) AS active_days GROUP BY grp) AS streaks), 0)::int,
    ((
        SELECT
            COUNT(*)
        FROM comments
        WHERE
            user_id = $1
            AND status = 'approved') + (
            SELECT
                COUNT(*)
            FROM comment_answers
            WHERE
                user_id = $1
                AND status = 'approved'))::int,
    COALESCE((
        SELECT
            SUM(cv.value)
        FROM comment_votes cv
        JOIN comments c ON cv.comment_id = c.id
        WHERE
            c.user_id = $1
            AND c.status = 'approved'), 0)::int;

