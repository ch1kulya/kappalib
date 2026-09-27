SELECT
    c.id,
    COALESCE(c.chapter_id, ''),
    c.user_id,
    c.content_html,
    c.status,
    c.edited_at,
    c.created_at,
    u.display_name,
    u.avatar_seed,
    u.has_custom_avatar,
    u.avatar_updated_at,
    COALESCE((
        SELECT
            SUM(value)
        FROM comment_votes
        WHERE
            comment_id = c.id), 0)::int,
    COALESCE((
        SELECT
            value
        FROM comment_votes
        WHERE
            comment_id = c.id
            AND user_id = $2), 0)::int,
    COALESCE(ch.chapter_num, 0),
    COALESCE(ch.novel_id, ''),
    COALESCE(n.title, ''),
    COALESCE(c.profile_id, ''),
    COALESCE(pu.display_name, ''),
    c.approved_at
FROM
    comments c
    JOIN users u ON c.user_id = u.id
    LEFT JOIN chapters ch ON c.chapter_id = ch.id
    LEFT JOIN novels n ON ch.novel_id = n.id
    LEFT JOIN users pu ON c.profile_id = pu.id
WHERE
    c.id = ANY ($1);

