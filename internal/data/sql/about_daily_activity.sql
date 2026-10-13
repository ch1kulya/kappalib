SELECT
    to_char(date, 'YYYY-MM-DD') AS day,
    coalesce(sum(seconds_spent), 0)::bigint AS total_seconds,
    count(distinct user_id)::int AS active_users
FROM
    user_daily_time
WHERE
    date >= CURRENT_DATE - 30
GROUP BY
    date
ORDER BY
    date ASC;