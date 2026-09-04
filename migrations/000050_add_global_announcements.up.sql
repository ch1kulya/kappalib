CREATE TABLE global_announcements (
    id serial PRIMARY KEY,
    text text NOT NULL,
    url text,
    is_active boolean NOT NULL DEFAULT TRUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

