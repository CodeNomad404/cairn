ALTER TABLE course_registration_credentials ADD COLUMN binding_version BIGINT NOT NULL DEFAULT 1;

CREATE TABLE course_homework_sync (
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    term_code TEXT NOT NULL,
    course_id TEXT NOT NULL,
    course_name TEXT NOT NULL,
    binding_version BIGINT NOT NULL,
    last_attempt_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_success_at TIMESTAMPTZ,
    error TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (user_id, term_code, course_id)
);

CREATE TABLE course_homework (
    id BIGSERIAL PRIMARY KEY,
    user_id BIGINT NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    term_code TEXT NOT NULL,
    course_id TEXT NOT NULL,
    platform_id BIGINT NOT NULL,
    binding_version BIGINT NOT NULL,
    payload JSONB NOT NULL,
    UNIQUE (user_id, term_code, course_id, platform_id)
);
