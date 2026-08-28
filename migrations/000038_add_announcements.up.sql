CREATE TABLE announcements (
    id serial PRIMARY KEY,
    title text NOT NULL,
    description text NOT NULL,
    action_label text,
    action_url text,
    is_active boolean NOT NULL DEFAULT TRUE,
    created_at timestamptz NOT NULL DEFAULT now()
);

