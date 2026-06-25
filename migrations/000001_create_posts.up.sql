CREATE TABLE posts (
    id            BIGSERIAL    PRIMARY KEY,
    slug          VARCHAR(255) NOT NULL UNIQUE,
    title         VARCHAR(255) NOT NULL,
    content_md    TEXT         NOT NULL DEFAULT '',
    content_html  TEXT         NOT NULL DEFAULT '',
    status        VARCHAR(32)  NOT NULL DEFAULT 'draft',
    created_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    published_at  TIMESTAMPTZ
);

CREATE INDEX idx_posts_slug    ON posts (slug);
CREATE INDEX idx_posts_status  ON posts (status);
CREATE INDEX idx_posts_created ON posts (created_at DESC);
