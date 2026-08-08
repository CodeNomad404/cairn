package model

import "time"

// Post 对应数据库 posts 表的一条记录
type Post struct {
	ID          int64
	Slug        string
	Title       string
	ContentMD   string
	ContentHTML string
	Status      string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	PublishedAt *time.Time
}
