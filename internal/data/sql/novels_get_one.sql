SELECT id, title, title_en, author, year_start, year_end, status,
       description, age_rating, cover_url, created_at
FROM novels WHERE id = $1;
