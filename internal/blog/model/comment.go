package model

import "time"

// Comment 对应数据库 comments 表的一条记录
type Comment struct {
	ID        int64
	PostID    int64
	ParentID  *int64
	UserID    int64
	Content   string
	Status    string
	CreatedAt time.Time
	// Username 和 Replies 仅查询时填充，不存 comments 表。
	Username string
	Replies  []Comment
}
