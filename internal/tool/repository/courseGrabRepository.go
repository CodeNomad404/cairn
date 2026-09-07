package repository

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
)

type courseGrabRepo struct {
	db *pgxpool.Pool
}

func NewCourseGrabRepo(db *pgxpool.Pool) tool.CourseGrabRepo {
	return &courseGrabRepo{db: db}
}

func (r *courseGrabRepo) RecordLoginAttempt(ctx context.Context, userID int64) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO course_grab_jobs (user_id, login_count, last_login_at)
		VALUES ($1, 1, NOW())
		ON CONFLICT (user_id) DO UPDATE SET
			login_count = course_grab_jobs.login_count + 1,
			last_login_at = NOW(),
			updated_at = NOW()`, userID)
	return err
}

func (r *courseGrabRepo) SetWorkerStatus(ctx context.Context, userID int64, status string) error {
	_, err := r.db.Exec(ctx, `
		INSERT INTO course_grab_jobs (user_id, worker_status)
		VALUES ($1, $2)
		ON CONFLICT (user_id) DO UPDATE SET
			worker_status = EXCLUDED.worker_status,
			updated_at = NOW()`, userID, status)
	return err
}

func (r *courseGrabRepo) TouchPoll(ctx context.Context, userID int64, nextPollAt time.Time) error {
	_, err := r.db.Exec(ctx, `
		UPDATE course_grab_jobs
		SET last_polled_at = NOW(), next_poll_at = $2, updated_at = NOW()
		WHERE user_id = $1`, userID, nextPollAt)
	return err
}

func (r *courseGrabRepo) ListRunnableUserIDs(ctx context.Context) ([]int64, error) {
	rows, err := r.db.Query(ctx, `
		SELECT DISTINCT j.user_id
		FROM course_grab_jobs AS j
		JOIN course_grab_items AS i ON i.job_id = j.id
		WHERE i.status = 'grabbing' AND j.worker_status <> 'auth_failed'
		ORDER BY j.user_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var userIDs []int64
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, userID)
	}
	return userIDs, rows.Err()
}

func (r *courseGrabRepo) UpsertItem(ctx context.Context, item *toolmodel.CourseGrabItem) error {
	return r.db.QueryRow(ctx, `
		WITH job AS (
			INSERT INTO course_grab_jobs (user_id)
			VALUES ($1)
			ON CONFLICT (user_id) DO UPDATE SET
				updated_at = NOW()
			RETURNING id
		)
		INSERT INTO course_grab_items (
			job_id, course_name, upstream_course_id, teacher, remaining, status, message
		)
		SELECT id, $2, NULLIF($3, ''), $4, $5, 'grabbing', $6 FROM job
		ON CONFLICT (job_id, course_name) DO UPDATE SET
			upstream_course_id = EXCLUDED.upstream_course_id,
			teacher = EXCLUDED.teacher,
			remaining = EXCLUDED.remaining,
			updated_at = NOW()
		RETURNING id, status, message, attempt_count, last_checked_at,
		          succeeded_at, created_at, updated_at`,
		item.UserID, item.CourseName, item.UpstreamCourseID, item.Teacher,
		item.Remaining, item.Message,
	).Scan(
		&item.ID, &item.Status, &item.Message, &item.AttemptCount,
		&item.LastCheckedAt, &item.SucceededAt, &item.CreatedAt, &item.UpdatedAt,
	)
}

func (r *courseGrabRepo) ListItems(ctx context.Context, userID int64) ([]toolmodel.CourseGrabItem, error) {
	return r.listItems(ctx, userID, "")
}

func (r *courseGrabRepo) ListItemsByStatus(
	ctx context.Context,
	userID int64,
	status string,
) ([]toolmodel.CourseGrabItem, error) {
	return r.listItems(ctx, userID, status)
}

func (r *courseGrabRepo) listItems(
	ctx context.Context,
	userID int64,
	status string,
) ([]toolmodel.CourseGrabItem, error) {
	rows, err := r.db.Query(ctx, `
		SELECT i.id, j.user_id, i.course_name, COALESCE(i.upstream_course_id, ''),
		       i.teacher, i.remaining, i.status, i.message, i.attempt_count,
		       i.last_checked_at, i.succeeded_at, i.created_at, i.updated_at
		FROM course_grab_items AS i
		JOIN course_grab_jobs AS j ON j.id = i.job_id
		WHERE j.user_id = $1 AND ($2 = '' OR i.status = $2)
		ORDER BY i.id`, userID, status)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := make([]toolmodel.CourseGrabItem, 0)
	for rows.Next() {
		var item toolmodel.CourseGrabItem
		if err := scanCourseGrabItem(rows, &item); err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *courseGrabRepo) GetItem(
	ctx context.Context,
	userID int64,
	itemID int64,
) (*toolmodel.CourseGrabItem, error) {
	item := &toolmodel.CourseGrabItem{}
	err := scanCourseGrabItem(r.db.QueryRow(ctx, `
		SELECT i.id, j.user_id, i.course_name, COALESCE(i.upstream_course_id, ''),
		       i.teacher, i.remaining, i.status, i.message, i.attempt_count,
		       i.last_checked_at, i.succeeded_at, i.created_at, i.updated_at
		FROM course_grab_items AS i
		JOIN course_grab_jobs AS j ON j.id = i.job_id
		WHERE j.user_id = $1 AND i.id = $2`, userID, itemID), item)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, tool.ErrCourseGrabItemNotFound
	}
	if err != nil {
		return nil, err
	}
	return item, nil
}

func (r *courseGrabRepo) UpdateItem(ctx context.Context, item *toolmodel.CourseGrabItem) error {
	command, err := r.db.Exec(ctx, `
		UPDATE course_grab_items AS i
		SET upstream_course_id = NULLIF($3, ''), teacher = $4, remaining = $5,
		    status = $6, message = $7, attempt_count = attempt_count + 1,
		    last_checked_at = NOW(),
		    succeeded_at = CASE WHEN $6 = 'succeeded' THEN COALESCE(succeeded_at, NOW()) ELSE succeeded_at END,
		    updated_at = NOW()
		FROM course_grab_jobs AS j
		WHERE i.job_id = j.id AND j.user_id = $1 AND i.id = $2
		  AND (i.status = 'grabbing' OR $6 = 'succeeded')`,
		item.UserID, item.ID, item.UpstreamCourseID, item.Teacher,
		item.Remaining, item.Status, item.Message,
	)
	if err != nil {
		return err
	}
	if command.RowsAffected() == 0 {
		return tool.ErrCourseGrabItemNotFound
	}
	return nil
}

func (r *courseGrabRepo) SetItemStatus(
	ctx context.Context,
	userID int64,
	itemID int64,
	status string,
	message string,
) (*toolmodel.CourseGrabItem, error) {
	command, err := r.db.Exec(ctx, `
		UPDATE course_grab_items AS i
		SET status = $3, message = $4, updated_at = NOW()
		FROM course_grab_jobs AS j
		WHERE i.job_id = j.id AND j.user_id = $1 AND i.id = $2
		  AND i.status IN ('grabbing', 'paused')`, userID, itemID, status, message)
	if err != nil {
		return nil, err
	}
	if command.RowsAffected() == 0 {
		return nil, tool.ErrCourseGrabItemNotFound
	}
	return r.GetItem(ctx, userID, itemID)
}

func (r *courseGrabRepo) GetJobSummary(
	ctx context.Context,
	userID int64,
) (toolmodel.CourseGrabJobSummary, error) {
	var summary toolmodel.CourseGrabJobSummary
	err := r.db.QueryRow(ctx, `
		SELECT worker_status, login_count, last_login_at
		FROM course_grab_jobs
		WHERE user_id = $1`, userID).Scan(
		&summary.WorkerStatus, &summary.LoginCount, &summary.LastLoginAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return toolmodel.CourseGrabJobSummary{WorkerStatus: "idle"}, nil
	}
	return summary, err
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanCourseGrabItem(row rowScanner, item *toolmodel.CourseGrabItem) error {
	return row.Scan(
		&item.ID, &item.UserID, &item.CourseName, &item.UpstreamCourseID,
		&item.Teacher, &item.Remaining, &item.Status, &item.Message,
		&item.AttemptCount, &item.LastCheckedAt, &item.SucceededAt,
		&item.CreatedAt, &item.UpdatedAt,
	)
}
