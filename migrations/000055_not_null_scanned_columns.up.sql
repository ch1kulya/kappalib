UPDATE
    users
SET
    created_at = COALESCE(last_active_at, now())
WHERE
    created_at IS NULL;

UPDATE
    users
SET
    avatar_updated_at = created_at
WHERE
    avatar_updated_at IS NULL;

UPDATE
    users
SET
    has_custom_avatar = FALSE
WHERE
    has_custom_avatar IS NULL;

UPDATE
    novels AS n
SET
    created_at = COALESCE((
        SELECT
            MIN(c.created_at)
        FROM chapters AS c
        WHERE
            c.novel_id = n.id), (
            SELECT
                MIN(o.created_at)
            FROM novels AS o), now())
WHERE
    n.created_at IS NULL;

UPDATE
    novels AS n
SET
    chapters_count = (
        SELECT
            COUNT(*)
        FROM
            chapters AS c
        WHERE
            c.novel_id = n.id)
WHERE
    n.chapters_count IS NULL;

UPDATE
    chapters AS c
SET
    created_at = n.created_at
FROM
    novels AS n
WHERE
    c.novel_id = n.id
    AND c.created_at IS NULL;

UPDATE
    comments AS c
SET
    created_at = COALESCE(c.approved_at, c.edited_at, u.created_at)
FROM
    users AS u
WHERE
    c.user_id = u.id
    AND c.created_at IS NULL;

UPDATE
    comment_answers AS a
SET
    created_at = COALESCE(a.approved_at, a.edited_at, c.created_at)
FROM
    comments AS c
WHERE
    a.comment_id = c.id
    AND a.created_at IS NULL;

ALTER TABLE users
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN avatar_updated_at SET NOT NULL,
    ALTER COLUMN has_custom_avatar SET NOT NULL;

ALTER TABLE novels
    ALTER COLUMN created_at SET NOT NULL,
    ALTER COLUMN chapters_count SET NOT NULL;

ALTER TABLE chapters
    ALTER COLUMN created_at SET NOT NULL;

ALTER TABLE comments
    ALTER COLUMN created_at SET NOT NULL;

ALTER TABLE comment_answers
    ALTER COLUMN created_at SET NOT NULL;

DO $$
BEGIN
    IF EXISTS (
        SELECT
            1
        FROM
            chapter_groups
        WHERE
            updated_at IS NULL) THEN
        REFRESH MATERIALIZED VIEW CONCURRENTLY chapter_groups;
    END IF;
END;
$$;

