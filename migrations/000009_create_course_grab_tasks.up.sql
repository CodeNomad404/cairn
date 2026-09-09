CREATE TABLE course_grab_jobs (
    id             BIGSERIAL PRIMARY KEY,
    user_id        BIGINT      NOT NULL UNIQUE REFERENCES users(id) ON DELETE CASCADE,
    worker_status  VARCHAR(16) NOT NULL DEFAULT 'idle'
        CHECK (worker_status IN ('idle', 'running', 'auth_failed')),
    login_count    INTEGER     NOT NULL DEFAULT 0,
    last_login_at  TIMESTAMPTZ,
    last_polled_at TIMESTAMPTZ,
    next_poll_at   TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE course_grab_items (
    id                   BIGSERIAL PRIMARY KEY,
    job_id               BIGINT       NOT NULL REFERENCES course_grab_jobs(id) ON DELETE CASCADE,
    course_name          VARCHAR(256) NOT NULL,
    upstream_course_id   TEXT,
    teacher              VARCHAR(128) NOT NULL DEFAULT '',
    remaining            INTEGER      NOT NULL DEFAULT 0,
    status               VARCHAR(16)  NOT NULL
        CHECK (status IN ('grabbing', 'paused', 'succeeded', 'not_found')),
    message              TEXT         NOT NULL DEFAULT '',
    attempt_count        INTEGER      NOT NULL DEFAULT 0,
    last_checked_at      TIMESTAMPTZ,
    succeeded_at         TIMESTAMPTZ,
    created_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at           TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    UNIQUE (job_id, course_name)
);

CREATE INDEX idx_course_grab_items_job_status_id
    ON course_grab_items (job_id, status, id);
