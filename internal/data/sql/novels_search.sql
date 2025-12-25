WITH norm_query AS (
    SELECT lower(regexp_replace($1, '[^[:alnum:]]', '', 'g')) AS q
)
SELECT
    n.id, n.title, n.title_en, n.author, n.year_start, n.year_end, n.status,
    n.description, n.age_rating, n.cover_url, n.created_at,
    (
        (word_similarity(nq.q, n.title_norm) * 2.5) +
        (word_similarity(nq.q, n.title_en_norm) * 2.0) +
        (similarity(n.author_norm, nq.q) * 1.0)
    ) as relevance
FROM novels n, norm_query nq
WHERE
    (nq.q <% n.title_norm) OR
    (nq.q <% n.title_en_norm) OR
    (n.author_norm % nq.q)
ORDER BY relevance DESC, n.created_at DESC
LIMIT 20;
