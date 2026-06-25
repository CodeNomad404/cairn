DROP INDEX IF EXISTS idx_comments_user_id;
ALTER TABLE comments DROP CONSTRAINT IF EXISTS fk_comments_user;
ALTER TABLE comments DROP COLUMN IF EXISTS user_id;
