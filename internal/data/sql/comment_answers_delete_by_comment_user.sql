UPDATE comment_answers
SET status = 'deleted'
WHERE comment_id = $1
  AND user_id = $2
  AND status != 'deleted';
