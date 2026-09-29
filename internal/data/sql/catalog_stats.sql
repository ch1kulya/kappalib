SELECT
    (
        SELECT
            count(*)
        FROM
            novels),
    (
        SELECT
            coalesce(sum(chapters_count), 0)::bigint
        FROM
            novels),
    (
        SELECT
            count(*)
        FROM
            sources
        WHERE
            chapters_count > 0), (
        SELECT
            coalesce(sum(characters_count), 0)::bigint
        FROM
            sources);

