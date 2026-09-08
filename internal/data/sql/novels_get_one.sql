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
    alt_titles,
    (
        SELECT
            max(created_at)
        FROM
            chapters
        WHERE
            novel_id = novels.id) AS last_chapter_at
FROM
    novels
WHERE
    id = $1;

