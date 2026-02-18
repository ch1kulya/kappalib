SELECT COUNT(*)
FROM comments
WHERE user_id = $1
  AND status != 'deleted';
