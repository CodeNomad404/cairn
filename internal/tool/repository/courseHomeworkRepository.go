package repository

import (
	"context"
	"encoding/json"
	"errors"

	"cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type HomeworkRepository struct{ db *pgxpool.Pool }

func NewHomeworkRepo(db *pgxpool.Pool) *HomeworkRepository { return &HomeworkRepository{db: db} }

// Hold the binding row until commit so a concurrent rebind cannot publish old-account data.
func lockHomeworkBinding(ctx context.Context, tx pgx.Tx, userID, version int64) error {
	var found int64
	err := tx.QueryRow(ctx, `SELECT binding_version FROM course_registration_credentials WHERE user_id=$1 AND binding_version=$2 FOR SHARE`, userID, version).Scan(&found)
	if errors.Is(err, pgx.ErrNoRows) {
		return tool.ErrMISBindingChanged
	}
	return err
}

func (r *HomeworkRepository) Prepare(ctx context.Context, userID, version int64, term string, courses []toolmodel.HomeworkCourse) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = lockHomeworkBinding(ctx, tx, userID, version); err != nil {
		return err
	}
	ids := []string{""}
	// The empty course is the run marker, also representing a successful empty course list.
	all := append([]toolmodel.HomeworkCourse{{}}, courses...)
	for _, c := range all {
		ids = append(ids, c.ID)
		_, err = tx.Exec(ctx, `INSERT INTO course_homework_sync (user_id,term_code,course_id,course_name,binding_version,error)
		VALUES ($1,$2,$3,$4,$5,'同步未完成') ON CONFLICT (user_id,term_code,course_id) DO UPDATE SET
		course_name=EXCLUDED.course_name, binding_version=EXCLUDED.binding_version,last_attempt_at=NOW(),error=EXCLUDED.error,
		last_success_at=CASE WHEN course_homework_sync.binding_version=EXCLUDED.binding_version THEN course_homework_sync.last_success_at END`, userID, term, c.ID, c.Name, version)
		if err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `DELETE FROM course_homework WHERE user_id=$1 AND term_code=$2 AND NOT(course_id=ANY($3))`, userID, term, ids); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM course_homework_sync WHERE user_id=$1 AND term_code=$2 AND NOT(course_id=ANY($3))`, userID, term, ids); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *HomeworkRepository) SaveCourse(ctx context.Context, userID, version int64, term string, course toolmodel.HomeworkCourse, items []toolmodel.Homework, syncError string) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = lockHomeworkBinding(ctx, tx, userID, version); err != nil {
		return err
	}
	if syncError == "" && course.ID != "" {
		ids := make([]int64, 0, len(items))
		for _, item := range items {
			payload, err := json.Marshal(item)
			if err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `INSERT INTO course_homework (user_id,term_code,course_id,platform_id,binding_version,payload)
			VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (user_id,term_code,course_id,platform_id) DO UPDATE SET binding_version=EXCLUDED.binding_version,payload=EXCLUDED.payload`, userID, term, course.ID, item.PlatformID, version, payload)
			if err != nil {
				return err
			}
			ids = append(ids, item.PlatformID)
		}
		if _, err = tx.Exec(ctx, `DELETE FROM course_homework WHERE user_id=$1 AND term_code=$2 AND course_id=$3 AND NOT(platform_id=ANY($4))`, userID, term, course.ID, ids); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE course_homework_sync SET last_attempt_at=NOW(),error=$5,
	last_success_at=CASE WHEN $5='' THEN NOW() ELSE last_success_at END
	WHERE user_id=$1 AND term_code=$2 AND course_id=$3 AND binding_version=$4`, userID, term, course.ID, version, syncError)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *HomeworkRepository) Snapshot(ctx context.Context, userID int64) (toolmodel.HomeworkSnapshot, error) {
	s := toolmodel.HomeworkSnapshot{Courses: []toolmodel.HomeworkCourse{}, Items: []toolmodel.Homework{}}
	err := r.db.QueryRow(ctx, `SELECT COALESCE(MAX(s.term_code),'') FROM course_homework_sync s JOIN course_registration_credentials c ON c.user_id=s.user_id AND c.binding_version=s.binding_version WHERE s.user_id=$1`, userID).Scan(&s.TermCode)
	if err != nil || s.TermCode == "" {
		return s, err
	}
	rows, err := r.db.Query(ctx, `SELECT s.course_id,s.course_name,s.last_attempt_at,s.last_success_at,s.error FROM course_homework_sync s JOIN course_registration_credentials c ON c.user_id=s.user_id AND c.binding_version=s.binding_version WHERE s.user_id=$1 AND s.term_code=$2 ORDER BY s.course_name,s.course_id`, userID, s.TermCode)
	if err != nil {
		return s, err
	}
	for rows.Next() {
		var c toolmodel.HomeworkCourse
		if err = rows.Scan(&c.ID, &c.Name, &c.LastAttemptAt, &c.LastSuccessAt, &c.Error); err != nil {
			rows.Close()
			return s, err
		}
		if c.ID == "" {
			s.HasSnapshot = true
			s.LastSuccessAt = c.LastSuccessAt
		} else {
			s.Courses = append(s.Courses, c)
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return s, err
	}
	rows, err = r.db.Query(ctx, `SELECT h.id,h.payload FROM course_homework h JOIN course_registration_credentials c ON c.user_id=h.user_id AND c.binding_version=h.binding_version WHERE h.user_id=$1 AND h.term_code=$2`, userID, s.TermCode)
	if err != nil {
		return s, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int64
		var payload []byte
		var item toolmodel.Homework
		if err = rows.Scan(&id, &payload); err != nil {
			return s, err
		}
		if err = json.Unmarshal(payload, &item); err != nil {
			return s, err
		}
		item.ID = id
		s.Items = append(s.Items, item)
	}
	return s, rows.Err()
}

func (r *HomeworkRepository) Get(ctx context.Context, userID, id int64) (*toolmodel.Homework, error) {
	var payload []byte
	err := r.db.QueryRow(ctx, `SELECT h.payload FROM course_homework h JOIN course_registration_credentials c ON c.user_id=h.user_id AND c.binding_version=h.binding_version WHERE h.user_id=$1 AND h.id=$2`, userID, id).Scan(&payload)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, tool.ErrHomeworkNotFound
	}
	if err != nil {
		return nil, err
	}
	var item toolmodel.Homework
	if err = json.Unmarshal(payload, &item); err != nil {
		return nil, err
	}
	item.ID = id
	return &item, nil
}
