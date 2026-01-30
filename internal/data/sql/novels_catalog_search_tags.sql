WITH tokens AS (
    SELECT unnest(
        string_to_array(
            lower(regexp_replace($1, '[^[:alnum:] ]', '', 'g')), ' '
        )
    ) AS token
),

filtered AS (
    SELECT token FROM tokens
    WHERE token <> ''
),

token_count AS (
    SELECT count(*) AS cnt FROM filtered
),

tag_matches AS (
    SELECT
        t.name,
        ft.token
    FROM filtered AS ft
    INNER JOIN tags AS t ON ft.token = t.name_norm
),

tag_count AS (
    SELECT count(*) AS cnt FROM tag_matches
)

SELECT coalesce(array_agg(tm.name ORDER BY tm.name), '{}')
FROM tag_matches AS tm, token_count AS tc, tag_count AS tgc
WHERE tc.cnt > 0 AND tc.cnt = tgc.cnt;
