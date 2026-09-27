SELECT
    COUNT(*)
FROM (
    SELECT
        id
    FROM
        comments
    WHERE
        user_id = $1
        AND status != 'deleted'
    UNION
    SELECT
        comment_id
    FROM
        comment_answers
    WHERE
        user_id = $1
        AND status != 'deleted'
    UNION
    SELECT
        id
    FROM
        comments
    WHERE
        profile_id = $1
        AND user_id != $1
        AND status = 'approved') AS combined;

