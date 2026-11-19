package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"cairn.cn/internal/auth"
	authmodel "cairn.cn/internal/auth/model"
)

type userRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) auth.UserRepo {
	return &userRepo{db: db}
}

func (r *userRepo) CreateWithSession(ctx context.Context, u *authmodel.User, s *authmodel.Session) error {
	err := r.db.QueryRow(ctx, `
		WITH new_user AS (
			INSERT INTO users (username, password, role) VALUES ($1, $2, $3)
			ON CONFLICT (username) DO NOTHING
			RETURNING id, created_at
		), new_session AS (
			INSERT INTO sessions (token, user_id, expires_at)
			SELECT $4, id, $5 FROM new_user
			RETURNING user_id
		)
		SELECT id, created_at FROM new_user JOIN new_session ON user_id = id`,
		u.Username, u.Password, u.Role, s.Token, s.ExpiresAt,
	).Scan(&u.ID, &u.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return auth.ErrUsernameTaken
	}
	return err
}

func (r *userRepo) GetByUsername(ctx context.Context, username string) (*authmodel.User, error) {
	u := &authmodel.User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, username, password, role, created_at FROM users WHERE username = $1`,
		username,
	).Scan(&u.ID, &u.Username, &u.Password, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (r *userRepo) GetByID(ctx context.Context, id int64) (*authmodel.User, error) {
	u := &authmodel.User{}
	err := r.db.QueryRow(ctx,
		`SELECT id, username, password, role, created_at FROM users WHERE id = $1`,
		id,
	).Scan(&u.ID, &u.Username, &u.Password, &u.Role, &u.CreatedAt)
	if err != nil {
		return nil, err
	}
	return u, nil
}
