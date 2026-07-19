package blog

import (
	"context"
	"errors"
	"time"
)

var ErrPostNotFound = errors.New("post not found")

// Post 对应数据库 posts 表的一条记录
type Post struct {
	ID          int64      `json:"id"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	ContentMD   string     `json:"content_md"`
	ContentHTML string     `json:"content_html"`
	Status      string     `json:"status"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	PublishedAt *time.Time `json:"published_at"`
}

// PostRepo 定义文章数据访问层必须提供的方法
type PostRepo interface {
	Create(ctx context.Context, p *Post) error
	GetBySlug(ctx context.Context, slug string) (*Post, error)
	ListPublished(ctx context.Context, limit, offset int) ([]Post, error)
	ListPublishedMetadata(ctx context.Context) ([]Post, error)
}

// Comment 对应数据库 comments 表的一条记录
type Comment struct {
	ID        int64     `json:"id"`
	PostID    int64     `json:"post_id"`
	ParentID  *int64    `json:"parent_id"`
	UserID    int64     `json:"user_id"`
	Content   string    `json:"content"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	// Username 和 Replies 仅查询时填充，不存 comments 表。
	Username string    `json:"username"`
	Replies  []Comment `json:"replies,omitempty"`
}

// CommentRepo 定义评论数据访问层必须提供的方法
type CommentRepo interface {
	Create(ctx context.Context, c *Comment) error
	GetByID(ctx context.Context, id int64) (*Comment, error)
	ListByPostID(ctx context.Context, postID int64) ([]Comment, error)
	Delete(ctx context.Context, id int64) error
}
