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
			item.Date, item.Period, item.BuildingID, item.BuildingName,
			item.RoomName, item.Capacity, item.OccupationType, item.SyncedAt,
		})
	}

	if _, err := tx.Exec(ctx, "DELETE FROM classroom_availability"); err != nil {
		return err
	}
	if _, err := tx.CopyFrom(
		ctx,
		pgx.Identifier{"classroom_availability"},
		[]string{
			"observed_date", "period", "building_id", "building_name",
			"room_name", "capacity", "occupation_type", "synced_at",
		},
		pgx.CopyFromRows(rows),
	); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func sameDate(left, right time.Time) bool {
	leftYear, leftMonth, leftDay := left.Date()
	rightYear, rightMonth, rightDay := right.Date()
	return leftYear == rightYear && leftMonth == rightMonth && leftDay == rightDay
}

func (r *classroomAvailabilityRepo) List(
	ctx context.Context,
	query tool.ClassroomAvailabilityQuery,
) ([]toolmodel.ClassroomAvailability, *time.Time, error) {
	rows, err := r.db.Query(ctx, `
		SELECT observed_date, period, building_id, building_name, room_name,
		       capacity, occupation_type, synced_at
		FROM classroom_availability
		WHERE observed_date = $1 AND ($2 = 0 OR period = $2)
		  AND ($3 = '' OR building_id = $3) AND capacity >= $4
		ORDER BY period, building_name, room_name`,
		query.Date, query.Period, query.BuildingID, query.MinCapacity,
	)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	items := make([]toolmodel.ClassroomAvailability, 0)
	for rows.Next() {
		var item toolmodel.ClassroomAvailability
		if err := rows.Scan(
			&item.Date, &item.Period, &item.BuildingID, &item.BuildingName,
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
		"SELECT MAX(synced_at) FROM classroom_availability WHERE observed_date = $1",
		query.Date,
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
