WITH active_days AS (
    SELECT
        date
    FROM
        user_daily_time
    WHERE
        user_id = $1
        AND seconds_spent > 0
),
streaks AS (
    SELECT
        MAX(date) AS last_day,
        COUNT(*) AS days
    FROM (
        SELECT
            date,
            date - (ROW_NUMBER() OVER (ORDER BY date))::int AS grp
        FROM
            active_days) AS grouped
    GROUP BY
        grp
)
SELECT
    CURRENT_DATE,
    COALESCE((
        SELECT
            days
        FROM streaks
        WHERE
            last_day >= CURRENT_DATE - 1), 0)::int,
    ARRAY (
        SELECT
            date
        FROM
            active_days
        WHERE
            date >= date_trunc('week', CURRENT_DATE)::date);

