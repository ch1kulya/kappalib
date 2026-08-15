SELECT
    id,
    title,
    author,
    year_start,
    status,
    description,
    cover_url,
    created_at
FROM novels
WHERE
    NOT (
        has_self_harm
        OR has_drug_usage
        OR has_sexual_violence
        OR has_graphic_sex
    )
ORDER BY created_at DESC
LIMIT NULLIF($1, 0);
