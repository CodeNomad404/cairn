package model

import "time"

// Session 对应数据库 sessions 表
type Session struct {
	Token     string
	UserID    int64
	CreatedAt time.Time
	ExpiresAt time.Time
}
