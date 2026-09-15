CREATE TABLE classroom_availability (
    observed_date   DATE         NOT NULL,
    period          SMALLINT     NOT NULL CHECK (period BETWEEN 1 AND 7),
    building_id     VARCHAR(16)  NOT NULL,
    building_name   VARCHAR(128) NOT NULL,
    room_name       VARCHAR(128) NOT NULL,
    capacity        INTEGER      NOT NULL CHECK (capacity >= 0),
    occupation_type VARCHAR(16)  NOT NULL
        CHECK (occupation_type IN ('free', 'scheduled', 'rescheduled', 'exam', 'laboratory', 'other')),
    synced_at       TIMESTAMPTZ  NOT NULL,
    PRIMARY KEY (observed_date, period, building_id, room_name)
);

CREATE INDEX idx_classroom_availability_lookup
    ON classroom_availability (observed_date, period, building_id, capacity)
    WHERE occupation_type = 'free';
