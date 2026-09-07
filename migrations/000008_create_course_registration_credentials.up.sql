CREATE TABLE course_registration_credentials (
    user_id          BIGINT PRIMARY KEY REFERENCES users(id) ON DELETE CASCADE,
    student_id       VARCHAR(64)  NOT NULL,
    bjtu_password    TEXT         NOT NULL,
    tujian_username  VARCHAR(128) NOT NULL,
    tujian_password  TEXT         NOT NULL,
    created_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);
