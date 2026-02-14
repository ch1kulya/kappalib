SELECT id, title, description, action_label, action_url
FROM announcements
WHERE is_active = true
ORDER BY random()
LIMIT 1;
