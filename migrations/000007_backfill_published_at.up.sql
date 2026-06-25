UPDATE posts
SET published_at = created_at
WHERE status = 'published' AND published_at IS NULL;
