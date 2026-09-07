package model

import "time"

// CourseRegistrationCredentials 对应 course_registration_credentials 表。
// 当前开发阶段密码暂以明文保存，功能稳定后必须迁移为加密存储。
type CourseRegistrationCredentials struct {
	UserID         int64
	StudentID      string
	BJTUPassword   string
	TujianUsername string
	TujianPassword string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
