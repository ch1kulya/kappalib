WITH norm_query AS (
    SELECT
        lower(regexp_replace($1, '[^[:alnum:]]', '', 'g')) AS q,
        '%' || lower(regexp_replace($1, '[^[:alnum:]]', '', 'g')) || '%' AS q_like,
        string_to_array(lower(regexp_replace($1, '[^[:alnum:] ]', '', 'g')), ' ') AS tokens
),
non_empty_tokens AS (
    SELECT
        unnest(nq.tokens) AS token
    FROM
        norm_query AS nq
    WHERE
        nq.tokens IS NOT NULL
        AND array_length(nq.tokens, 1) > 0
),
filtered_tokens AS (
    SELECT
        net.token
    FROM
        non_empty_tokens AS net
    WHERE
        net.token <> ''
),
token_count AS (
    SELECT
        count(*) AS cnt
    FROM
        filtered_tokens
),
tag_token_count AS (
    SELECT
        count(*) AS cnt
    FROM
        filtered_tokens AS ft
    WHERE
        EXISTS (
            SELECT
                1
            FROM
                tags AS tg
            WHERE
                tg.name_norm = ft.token)
),
query_type AS (
    SELECT
        tc.cnt AS total_tokens,
        ttc.cnt AS tag_tokens,
        tc.cnt > 0
        AND tc.cnt = ttc.cnt AS is_tag_only
    FROM
        token_count AS tc,
        tag_token_count AS ttc
),
text_candidates AS (
    SELECT
        n.*,
        nq.q,
        nq.q_like
    FROM
        novels AS n,
        norm_query AS nq,
        query_type AS qt
    WHERE
        NOT qt.is_tag_only
        AND (n.title_norm ILIKE nq.q_like
            OR n.title_en_norm ILIKE nq.q_like
            OR n.author_norm ILIKE nq.q_like
            OR n.alt_titles_norm ILIKE nq.q_like
            OR nq.q % n.title_norm
            OR nq.q % n.title_en_norm
            OR nq.q % n.author_norm
            OR nq.q % n.alt_titles_norm
            OR nq.q <% n.title_norm
            OR nq.q <% n.title_en_norm
            OR nq.q <% n.alt_titles_norm
            OR (qt.total_tokens > 1
                AND NOT EXISTS (
                    SELECT
                        1
                    FROM
                        filtered_tokens AS ft
                    WHERE
                        NOT (n.title_norm ILIKE '%' || ft.token || '%'
                            OR n.title_en_norm ILIKE '%' || ft.token || '%'
                            OR n.author_norm ILIKE '%' || ft.token || '%'
                            OR n.alt_titles_norm ILIKE '%' || ft.token || '%'))))
),
tag_candidates AS (
    SELECT DISTINCT
        n.*,
        nq.q,
        nq.q_like
    FROM
        norm_query AS nq,
        query_type AS qt,
        novels AS n
    WHERE
        qt.is_tag_only
        AND NOT EXISTS (
            SELECT
                1
            FROM
                filtered_tokens AS ft
            WHERE
                NOT EXISTS (
                    SELECT
                        1
                    FROM
                        novel_tags AS nt
                        INNER JOIN tags AS tg ON nt.tag_id = tg.id
                    WHERE
                        nt.novel_id = n.id
                        AND tg.name_norm = ft.token))
),
candidates AS (
    SELECT
        *
    FROM
        text_candidates
    UNION ALL
    SELECT
        *
    FROM
        tag_candidates
)
SELECT
    c.id,
    c.title,
    c.title_en,
    c.author,
    c.year_start,
    c.year_end,
    c.status,
    c.description,
    c.age_rating,
    c.cover_url,
    c.created_at,
    c.chapters_count,
    c.has_self_harm,
    c.has_drug_usage,
    c.has_sexual_violence,
    c.has_graphic_sex,
    c.has_profanity,
    (
        CASE WHEN c.title_norm = c.q THEN
            100
        ELSE
            0
        END + CASE WHEN c.title_en_norm = c.q THEN
            100
        ELSE
            0
        END + CASE WHEN c.title_norm LIKE c.q || '%' THEN
            50
        ELSE
            0
        END + CASE WHEN c.title_en_norm LIKE c.q || '%' THEN
            50
        ELSE
            0
        END + CASE WHEN c.title_norm ILIKE c.q_like THEN
            25
        ELSE
            0
        END + CASE WHEN c.title_en_norm ILIKE c.q_like THEN
            20
        ELSE
            0
        END + CASE WHEN c.author_norm ILIKE c.q_like THEN
            15
        ELSE
            0
        END + CASE WHEN c.alt_titles_norm ILIKE c.q_like THEN
            20
        ELSE
            0
        END + similarity (c.q, c.title_norm) * 30 + similarity (c.q, c.title_en_norm) * 25 + similarity (c.q, c.author_norm) * 15 + similarity (c.q, c.alt_titles_norm) * 20 + word_similarity (c.q, c.title_norm) * 20 + word_similarity (c.q, c.title_en_norm) * 15 + word_similarity (c.q, c.alt_titles_norm) * 15) AS relevance
FROM
    candidates AS c
ORDER BY
    relevance DESC,
    c.created_at DESC
LIMIT 5;

