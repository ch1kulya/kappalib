UPDATE
    comment_answers
SET
    status = $1
WHERE
    id = $2
RETURNING
    id;

