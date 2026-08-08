package model

import "time"

// User 对应数据库 users 表
type User struct {
	ID        int64
	Username  string
	Password  string // bcrypt hash
	Role      string // admin | user
	CreatedAt time.Time
}
