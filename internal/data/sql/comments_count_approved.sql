SELECT COUNT(*) FROM comments WHERE chapter_id = $1 AND status = 'approved';
