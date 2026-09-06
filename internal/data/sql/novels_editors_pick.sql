SELECT n.id, n.title, n.title_en, n.author, n.year_start, n.year_end,
       n.status, n.description, n.age_rating, n.cover_url, n.created_at, n.chapters_count,
       n.has_self_harm, n.has_drug_usage, n.has_sexual_violence, n.has_graphic_sex, n.has_profanity
FROM editors_picks ep
JOIN novels n ON n.id = ep.novel_id
ORDER BY ep.position ASC
LIMIT 4
