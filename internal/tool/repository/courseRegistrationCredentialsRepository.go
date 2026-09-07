package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
)

type courseRegistrationCredentialsRepo struct {
	db *pgxpool.Pool
}

func NewCourseRegistrationCredentialsRepo(db *pgxpool.Pool) tool.CourseRegistrationCredentialsRepo {
	return &courseRegistrationCredentialsRepo{db: db}
}

func (r *courseRegistrationCredentialsRepo) GetByUserID(
	ctx context.Context,
	userID int64,
) (*toolmodel.CourseRegistrationCredentials, error) {
	credentials := &toolmodel.CourseRegistrationCredentials{}
	err := r.db.QueryRow(ctx, `
		SELECT user_id, student_id, bjtu_password, tujian_username, tujian_password, created_at, updated_at
		FROM course_registration_credentials
		WHERE user_id = $1`, userID).Scan(
		&credentials.UserID,
		&credentials.StudentID,
		&credentials.BJTUPassword,
		&credentials.TujianUsername,
		&credentials.TujianPassword,
		&credentials.CreatedAt,
		&credentials.UpdatedAt,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, tool.ErrCourseRegistrationCredentialsNotFound
	}
	if err != nil {
		return nil, err
	}
	return credentials, nil
}

func (r *courseRegistrationCredentialsRepo) Upsert(
	ctx context.Context,
	credentials *toolmodel.CourseRegistrationCredentials,
) error {
	return r.db.QueryRow(ctx, `
		INSERT INTO course_registration_credentials (
			user_id, student_id, bjtu_password, tujian_username, tujian_password
		) VALUES ($1, $2, $3, $4, $5)
		ON CONFLICT (user_id) DO UPDATE SET
			student_id = EXCLUDED.student_id,
			bjtu_password = EXCLUDED.bjtu_password,
			tujian_username = EXCLUDED.tujian_username,
			tujian_password = EXCLUDED.tujian_password,
			updated_at = NOW()
		RETURNING created_at, updated_at`,
		credentials.UserID,
		credentials.StudentID,
		credentials.BJTUPassword,
		credentials.TujianUsername,
		credentials.TujianPassword,
	).Scan(&credentials.CreatedAt, &credentials.UpdatedAt)
}
