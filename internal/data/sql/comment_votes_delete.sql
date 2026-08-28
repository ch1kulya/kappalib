DELETE FROM comment_votes
WHERE comment_id = $1
    AND user_id = $2;

