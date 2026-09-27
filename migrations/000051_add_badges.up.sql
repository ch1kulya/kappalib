CREATE TABLE IF NOT EXISTS badges (
    id varchar(50) PRIMARY KEY,
    title varchar(50) NOT NULL,
    description varchar(200) NOT NULL,
    icon varchar(30) NOT NULL DEFAULT 'award',
    sort_order integer NOT NULL DEFAULT 0,
    created_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS user_badges (
    user_id varchar(20) NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    badge_id varchar(50) NOT NULL REFERENCES badges (id) ON DELETE CASCADE,
    awarded_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, badge_id)
);

CREATE INDEX IF NOT EXISTS idx_user_badges_badge_id ON user_badges (badge_id);

INSERT INTO badges (id, title, description, icon, sort_order)
VALUES
    ('donor', 'Меценат', 'Поддержка проекта донатом', 'heart', 1),
    ('contributor', 'Соавтор', 'Вклад в развитие проекта', 'code', 2),
    ('stargazer', 'Звездочёт', 'Звезда проекту на GitHub', 'star', 3)
ON CONFLICT (id)
    DO NOTHING;

