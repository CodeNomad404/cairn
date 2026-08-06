package auth

import (
	"context"
	"time"
)

const (
	SessionCookie = "cairn_session"
	RoleAdmin     = "admin"
	RoleUser      = "user"
)

// User 对应数据库 users 表
type User struct {
	ID        int64     `json:"id"`
	Username  string    `json:"username"`
	Password  string    `json:"-"`    // bcrypt hash，不序列化
	Role      string    `json:"role"` // admin | user
	CreatedAt time.Time `json:"created_at"`
}

// Session 对应数据库 sessions 表
type Session struct {
	Token     string    `json:"token"`
	UserID    int64     `json:"user_id"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
}

// UserRepo 用户数据访问接口
type UserRepo interface {
	Create(ctx context.Context, u *User) error
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByID(ctx context.Context, id int64) (*User, error)
}

// SessionRepo 会话数据访问接口
type SessionRepo interface {
	Create(ctx context.Context, s *Session) error
	GetByToken(ctx context.Context, token string) (*Session, error)
	Delete(ctx context.Context, token string) error
	DeleteExpired(ctx context.Context) error
}
