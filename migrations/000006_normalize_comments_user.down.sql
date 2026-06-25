ALTER TABLE comments ADD COLUMN author VARCHAR(64);
ALTER TABLE comments ADD COLUMN email VARCHAR(128) NOT NULL DEFAULT '';

UPDATE comments AS c
SET author = u.username
FROM users AS u
WHERE c.user_id = u.id;

ALTER TABLE comments ALTER COLUMN author SET NOT NULL;
ALTER TABLE comments ALTER COLUMN email DROP DEFAULT;

ALTER TABLE comments DROP CONSTRAINT IF EXISTS fk_comments_user;
ALTER TABLE comments ALTER COLUMN user_id DROP NOT NULL;
ALTER TABLE comments
    ADD CONSTRAINT fk_comments_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;
