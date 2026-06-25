CREATE TABLE comments (
    id         BIGSERIAL PRIMARY KEY,
    post_id    BIGINT       NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
    parent_id  BIGINT       REFERENCES comments(id) ON DELETE CASCADE,
    author     VARCHAR(64)  NOT NULL,
    email      VARCHAR(128) NOT NULL,
    content    TEXT         NOT NULL,
    status     VARCHAR(16)  NOT NULL DEFAULT 'approved',
    created_at TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_comments_post_id ON comments(post_id, created_at);
CREATE INDEX idx_comments_parent_id ON comments(parent_id);
