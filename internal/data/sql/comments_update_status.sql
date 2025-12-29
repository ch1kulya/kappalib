UPDATE comments SET status = $1 WHERE id = $2 RETURNING id;
