SELECT
    EXISTS (
        SELECT
            1
        FROM
            novels
        WHERE
            id = $1);

