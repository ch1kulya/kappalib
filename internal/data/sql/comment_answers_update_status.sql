UPDATE
    comment_answers
SET
    status = $1::varchar,
    approved_at = CASE WHEN $1::varchar = 'approved' THEN
        COALESCE(approved_at, now())
    ELSE
        approved_at
    END
WHERE
    id = $2
RETURNING
    id,
    user_id;

