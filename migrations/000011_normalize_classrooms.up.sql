CREATE TABLE classrooms (
    id            BIGSERIAL    PRIMARY KEY,
    building_id   VARCHAR(16)  NOT NULL,
    building_name VARCHAR(128) NOT NULL,
    room_name     VARCHAR(128) NOT NULL,
    capacity      INTEGER      NOT NULL CHECK (capacity >= 0),
    UNIQUE (building_id, room_name)
);

INSERT INTO classrooms (building_id, building_name, room_name, capacity)
SELECT DISTINCT ON (building_id, room_name)
       building_id, building_name, room_name, capacity
FROM classroom_availability
ORDER BY building_id, room_name, synced_at DESC;

CREATE INDEX idx_classrooms_building_capacity
    ON classrooms (building_id, capacity);

CREATE TABLE classroom_availability_normalized (
    weekday         SMALLINT    NOT NULL CHECK (weekday BETWEEN 1 AND 7),
    period          SMALLINT    NOT NULL CHECK (period BETWEEN 1 AND 7),
    classroom_id    BIGINT      NOT NULL REFERENCES classrooms (id) ON DELETE CASCADE,
    occupation_type VARCHAR(16) NOT NULL
        CHECK (occupation_type IN ('free', 'scheduled', 'rescheduled', 'exam', 'laboratory', 'other')),
    synced_at       TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (weekday, period, classroom_id)
);

INSERT INTO classroom_availability_normalized (
    weekday, period, classroom_id, occupation_type, synced_at
)
SELECT DISTINCT ON (
           EXTRACT(ISODOW FROM availability.observed_date),
           availability.period,
           classrooms.id
       )
       EXTRACT(ISODOW FROM availability.observed_date)::SMALLINT,
       availability.period,
       classrooms.id,
       availability.occupation_type,
       availability.synced_at
FROM classroom_availability AS availability
JOIN classrooms
  ON classrooms.building_id = availability.building_id
 AND classrooms.room_name = availability.room_name
ORDER BY EXTRACT(ISODOW FROM availability.observed_date),
         availability.period,
         classrooms.id,
         availability.synced_at DESC;

DROP TABLE classroom_availability;
ALTER TABLE classroom_availability_normalized RENAME TO classroom_availability;

CREATE INDEX idx_classroom_availability_lookup
    ON classroom_availability (weekday, period, classroom_id)
    WHERE occupation_type = 'free';
