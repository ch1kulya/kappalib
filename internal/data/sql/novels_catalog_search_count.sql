WITH norm_query AS (
    SELECT
        lower(regexp_replace($1, '[^[:alnum:]]', '', 'g')) AS q,
        '%'
        || lower(regexp_replace($1, '[^[:alnum:]]', '', 'g'))
        || '%' AS q_like
)

SELECT count(*)
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
    OR nq.q <% n.alt_titles_norm;
