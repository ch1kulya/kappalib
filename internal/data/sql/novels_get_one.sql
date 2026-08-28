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
    alt_titles
FROM
    novels
WHERE
    id = $1;

