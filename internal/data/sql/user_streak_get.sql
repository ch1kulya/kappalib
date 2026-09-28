WITH streaks AS (
    SELECT
        MAX(date) AS last_day,
        COUNT(*) AS days
    FROM (
        SELECT
            date,
            date - (ROW_NUMBER() OVER (ORDER BY date))::int AS grp
        FROM
            user_daily_time
        WHERE
            user_id = $1
            AND seconds_spent > 0) AS active_days
    GROUP BY
        grp
)
SELECT
    COALESCE(MAX(days), 0)::int,
    COALESCE(BOOL_OR(last_day = CURRENT_DATE), FALSE)
FROM
    streaks
WHERE
    last_day >= CURRENT_DATE -1;

