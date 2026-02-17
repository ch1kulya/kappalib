SELECT COALESCE(SUM(value), 0)::int
FROM comment_votes
WHERE comment_id = $1;
