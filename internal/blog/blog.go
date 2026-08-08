package blog

import (
	"context"
	"errors"

	blogmodel "cairn.cn/internal/blog/model"
)

var ErrPostNotFound = errors.New("post not found")

// PostRepo 定义文章数据访问层必须提供的方法
type PostRepo interface {
	Create(ctx context.Context, post *blogmodel.Post) error
	GetBySlug(ctx context.Context, slug string) (*blogmodel.Post, error)
	ListPublished(ctx context.Context, limit, offset int) ([]blogmodel.Post, error)
	ListPublishedMetadata(ctx context.Context) ([]blogmodel.Post, error)
}

// CommentRepo 定义评论数据访问层必须提供的方法
type CommentRepo interface {
	Create(ctx context.Context, comment *blogmodel.Comment) error
	GetByID(ctx context.Context, id int64) (*blogmodel.Comment, error)
	ListByPostID(ctx context.Context, postID int64) ([]blogmodel.Comment, error)
	Delete(ctx context.Context, id int64) error
}
