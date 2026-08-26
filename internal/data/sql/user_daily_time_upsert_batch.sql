INSERT INTO user_daily_time (user_id, date, seconds_spent)
SELECT u, CURRENT_DATE, s
FROM UNNEST($1::text[], $2::int[]) AS t(u, s)
ON CONFLICT (user_id, date)
DO UPDATE SET seconds_spent = LEAST(86400, user_daily_time.seconds_spent + EXCLUDED.seconds_spent);
