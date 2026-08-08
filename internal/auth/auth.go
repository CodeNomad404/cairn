package auth

import (
	"context"

	authmodel "cairn.cn/internal/auth/model"
)

const (
	SessionCookie = "cairn_session"
	RoleAdmin     = "admin"
	RoleUser      = "user"
)

// UserRepo 用户数据访问接口
type UserRepo interface {
	Create(ctx context.Context, user *authmodel.User) error
	GetByUsername(ctx context.Context, username string) (*authmodel.User, error)
	GetByID(ctx context.Context, id int64) (*authmodel.User, error)
}

// SessionRepo 会话数据访问接口
type SessionRepo interface {
	Create(ctx context.Context, session *authmodel.Session) error
	GetByToken(ctx context.Context, token string) (*authmodel.Session, error)
	Delete(ctx context.Context, token string) error
	DeleteExpired(ctx context.Context) error
}
