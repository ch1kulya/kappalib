SELECT
    d.day::date,
    COALESCE(t.seconds_spent, 0)
FROM
    generate_series(date_trunc('week', CURRENT_DATE)::date - 364, CURRENT_DATE, INTERVAL '1 day') AS d (day)
    LEFT JOIN user_daily_time t ON t.user_id = $1
        AND t.date = d.day::date
    ORDER BY
        d.day;

