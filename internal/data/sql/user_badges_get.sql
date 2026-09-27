SELECT
    b.id,
    b.title,
    b.description,
    b.icon,
    ub.awarded_at
FROM
    user_badges ub
    JOIN badges b ON b.id = ub.badge_id
WHERE
    ub.user_id = $1
ORDER BY
    b.sort_order,
    ub.awarded_at;

