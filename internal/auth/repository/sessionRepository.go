package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"cairn.cn/internal/auth"
	authmodel "cairn.cn/internal/auth/model"
)

type sessionRepo struct {
	db *pgxpool.Pool
}

func NewSessionRepo(db *pgxpool.Pool) auth.SessionRepo {
	return &sessionRepo{db: db}
}

func (r *sessionRepo) Create(ctx context.Context, s *authmodel.Session) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO sessions (token, user_id, expires_at) VALUES ($1, $2, $3)`,
		s.Token, s.UserID, s.ExpiresAt,
	)
	return err
}

func (r *sessionRepo) GetByToken(ctx context.Context, token string) (*authmodel.Session, error) {
	s := &authmodel.Session{}
	err := r.db.QueryRow(ctx,
		`SELECT token, user_id, created_at, expires_at FROM sessions WHERE token = $1`,
		token,
	).Scan(&s.Token, &s.UserID, &s.CreatedAt, &s.ExpiresAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (r *sessionRepo) Delete(ctx context.Context, token string) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE token = $1`, token)
	return err
}

func (r *sessionRepo) DeleteExpired(ctx context.Context) error {
	_, err := r.db.Exec(ctx, `DELETE FROM sessions WHERE expires_at < NOW()`)
	return err
}
