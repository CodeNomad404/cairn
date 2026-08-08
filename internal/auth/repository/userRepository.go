package repository

import (
	"context"

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

func (r *userRepo) Create(ctx context.Context, u *authmodel.User) error {
	return r.db.QueryRow(ctx,
		`INSERT INTO users (username, password, role) VALUES ($1, $2, $3) RETURNING id, created_at`,
		u.Username, u.Password, u.Role,
	).Scan(&u.ID, &u.CreatedAt)
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
