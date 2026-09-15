CREATE TABLE classroom_availability_denormalized (
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

INSERT INTO classroom_availability_denormalized (
    observed_date, period, building_id, building_name,
    room_name, capacity, occupation_type, synced_at
)
SELECT (availability.synced_at AT TIME ZONE 'Asia/Shanghai')::DATE,
       availability.period,
       classrooms.building_id,
       classrooms.building_name,
       classrooms.room_name,
       classrooms.capacity,
       availability.occupation_type,
       availability.synced_at
FROM classroom_availability AS availability
JOIN classrooms ON classrooms.id = availability.classroom_id;

DROP TABLE classroom_availability;
ALTER TABLE classroom_availability_denormalized RENAME TO classroom_availability;
DROP TABLE classrooms;

CREATE INDEX idx_classroom_availability_lookup
    ON classroom_availability (observed_date, period, building_id, capacity)
    WHERE occupation_type = 'free';
