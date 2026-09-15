package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
)

const classroomSyncAdvisoryLock int64 = 2026091501

type classroomAvailabilityRepo struct {
	db *pgxpool.Pool
}

func NewClassroomAvailabilityRepo(db *pgxpool.Pool) tool.ClassroomAvailabilityRepo {
	return &classroomAvailabilityRepo{db: db}
}

func (r *classroomAvailabilityRepo) ReplaceSnapshot(
	ctx context.Context,
	items []toolmodel.ClassroomAvailability,
) error {
	if len(items) == 0 {
		return fmt.Errorf("classroom snapshot is empty")
	}

	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	var locked bool
	if err := tx.QueryRow(ctx, "SELECT pg_try_advisory_xact_lock($1)", classroomSyncAdvisoryLock).Scan(&locked); err != nil {
		return err
	}
	if !locked {
		return fmt.Errorf("classroom synchronization is already running")
	}

	snapshotDate := items[0].Date
	rows := make([][]any, 0, len(items))
	for _, item := range items {
		if !sameDate(item.Date, snapshotDate) {
			return fmt.Errorf("classroom snapshot contains multiple dates")
		}
		rows = append(rows, []any{
			isoWeekday(item.Date), item.Period, item.BuildingID, item.BuildingName,
			item.RoomName, item.Capacity, item.OccupationType, item.SyncedAt,
		})
	}

	if _, err := tx.Exec(ctx, `
		CREATE TEMP TABLE classroom_snapshot (
			weekday SMALLINT NOT NULL,
			period SMALLINT NOT NULL,
			building_id VARCHAR(16) NOT NULL,
			building_name VARCHAR(128) NOT NULL,
			room_name VARCHAR(128) NOT NULL,
			capacity INTEGER NOT NULL,
			occupation_type VARCHAR(16) NOT NULL,
			synced_at TIMESTAMPTZ NOT NULL
		) ON COMMIT DROP`); err != nil {
		return err
	}
	if _, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"classroom_snapshot"},
		[]string{
			"weekday", "period", "building_id", "building_name",
			"room_name", "capacity", "occupation_type", "synced_at",
		},
		pgx.CopyFromRows(rows),
	); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO classrooms (building_id, building_name, room_name, capacity)
		SELECT DISTINCT ON (building_id, room_name)
		       building_id, building_name, room_name, capacity
		FROM classroom_snapshot
		ORDER BY building_id, room_name, synced_at DESC
		ON CONFLICT (building_id, room_name) DO UPDATE SET
			building_name = EXCLUDED.building_name,
			capacity = EXCLUDED.capacity`); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO classroom_availability (
			weekday, period, classroom_id, occupation_type, synced_at
		)
		SELECT snapshot.weekday,
		       snapshot.period,
		       classrooms.id,
		       snapshot.occupation_type,
		       snapshot.synced_at
		FROM classroom_snapshot AS snapshot
		JOIN classrooms
		  ON classrooms.building_id = snapshot.building_id
		 AND classrooms.room_name = snapshot.room_name
		ON CONFLICT (weekday, period, classroom_id) DO UPDATE SET
			occupation_type = EXCLUDED.occupation_type,
			synced_at = EXCLUDED.synced_at`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func sameDate(left, right time.Time) bool {
	leftYear, leftMonth, leftDay := left.Date()
	rightYear, rightMonth, rightDay := right.Date()
	return leftYear == rightYear && leftMonth == rightMonth && leftDay == rightDay
}

func isoWeekday(value time.Time) int {
	weekday := int(value.Weekday())
	if weekday == 0 {
		return 7
	}
	return weekday
}

func (r *classroomAvailabilityRepo) List(
	ctx context.Context,
	query tool.ClassroomAvailabilityQuery,
) ([]toolmodel.ClassroomAvailability, *time.Time, error) {
	rows, err := r.db.Query(ctx, `
		SELECT availability.period,
		       classrooms.building_id,
		       classrooms.building_name,
		       classrooms.room_name,
		       classrooms.capacity,
		       availability.occupation_type,
		       availability.synced_at
		FROM classroom_availability AS availability
		JOIN classrooms ON classrooms.id = availability.classroom_id
		WHERE availability.weekday = $1
		  AND ($2 = 0 OR availability.period = $2)
		  AND ($3 = '' OR classrooms.building_id = $3)
		  AND classrooms.capacity >= $4
		ORDER BY availability.period, classrooms.building_name, classrooms.room_name`,
		query.Weekday, query.Period, query.BuildingID, query.MinCapacity,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	items := make([]toolmodel.ClassroomAvailability, 0)
	for rows.Next() {
		var item toolmodel.ClassroomAvailability
		if err := rows.Scan(
			&item.Period, &item.BuildingID, &item.BuildingName,
			&item.RoomName, &item.Capacity, &item.OccupationType, &item.SyncedAt,
		); err != nil {
			return nil, nil, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	var syncedAt *time.Time
	if err := r.db.QueryRow(ctx,
		"SELECT MAX(synced_at) FROM classroom_availability WHERE weekday = $1",
		query.Weekday,
	).Scan(&syncedAt); err != nil {
		return nil, nil, err
	}
	return items, syncedAt, nil
}

func (r *classroomAvailabilityRepo) LastSyncedAt(ctx context.Context) (*time.Time, error) {
	var syncedAt *time.Time
	err := r.db.QueryRow(ctx, "SELECT MAX(synced_at) FROM classroom_availability").Scan(&syncedAt)
	return syncedAt, err
}
