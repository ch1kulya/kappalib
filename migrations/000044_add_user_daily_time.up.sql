CREATE TABLE IF NOT EXISTS user_daily_time (
    user_id varchar(20) NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    date date NOT NULL DEFAULT CURRENT_DATE,
    seconds_spent integer NOT NULL DEFAULT 0,
    PRIMARY KEY (user_id, date)
);

CREATE INDEX IF NOT EXISTS idx_user_daily_time_date ON user_daily_time (date);

