SELECT
    novel_list
FROM
    users
WHERE
    id = $1
FOR UPDATE
