INSERT INTO badges (id, title, description, icon, sort_order)
    VALUES ('owner', 'Создатель', 'Создатель и владелец kappalib', 'crown', 0)
ON CONFLICT (id)
    DO NOTHING;

