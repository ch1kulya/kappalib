WITH norm_query AS (
    SELECT
        lower(regexp_replace($1, '[^[:alnum:]]', '', 'g')) AS q,
        '%'
        || lower(regexp_replace($1, '[^[:alnum:]]', '', 'g'))
        || '%' AS q_like
),

candidates AS (
    SELECT
        n.*,
        nq.q,
        nq.q_like
    FROM novels AS n, norm_query AS nq
    WHERE
        n.title_norm ILIKE nq.q_like
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
)

SELECT
    id,
    title,
    title_en,
    author,
    year_start,
    year_end,
    status,
    description,
    age_rating,
    cover_url,
    created_at,
    chapters_count,
    has_self_harm,
    has_drug_usage,
    has_sexual_violence,
    has_graphic_sex,
    has_profanity,
    (
        CASE WHEN title_norm = q THEN 100 ELSE 0 END
        + CASE WHEN title_en_norm = q THEN 100 ELSE 0 END

        + CASE WHEN title_norm LIKE q || '%' THEN 50 ELSE 0 END
        + CASE WHEN title_en_norm LIKE q || '%' THEN 50 ELSE 0 END

        + CASE WHEN title_norm ILIKE q_like THEN 25 ELSE 0 END
        + CASE WHEN title_en_norm ILIKE q_like THEN 20 ELSE 0 END
        + CASE WHEN author_norm ILIKE q_like THEN 15 ELSE 0 END
        + CASE WHEN alt_titles_norm ILIKE q_like THEN 20 ELSE 0 END

        + similarity(q, title_norm) * 30
        + similarity(q, title_en_norm) * 25
        + similarity(q, author_norm) * 15
        + similarity(q, alt_titles_norm) * 20

        + word_similarity(q, title_norm) * 20
        + word_similarity(q, title_en_norm) * 15
        + word_similarity(q, alt_titles_norm) * 15
    ) AS relevance
FROM candidates
ORDER BY relevance DESC, created_at DESC
LIMIT 5;
