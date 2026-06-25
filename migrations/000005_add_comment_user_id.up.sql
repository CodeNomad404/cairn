ALTER TABLE comments ADD COLUMN user_id BIGINT;

UPDATE comments AS c
SET user_id = u.id
FROM users AS u
WHERE c.author = u.username;

ALTER TABLE comments
    ADD CONSTRAINT fk_comments_user
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE SET NULL;

CREATE INDEX idx_comments_user_id ON comments(user_id);
