package logic

import (
	"context"
	"errors"
	"strings"

	"github.com/microcosm-cc/bluemonday"

	"cairn.cn/internal/blog"
)

// CommentLogic 评论相关的业务逻辑
type CommentLogic struct {
	commentRepo blog.CommentRepo
	policy      *bluemonday.Policy
}

// NewCommentLogic 创建 CommentLogic 实例
func NewCommentLogic(commentRepo blog.CommentRepo) *CommentLogic {
	// 评论只允许纯文本，比文章更严格
	policy := bluemonday.StrictPolicy()
	return &CommentLogic{
		commentRepo: commentRepo,
		policy:      policy,
	}
}

// Create 提交一条评论
func (s *CommentLogic) Create(ctx context.Context, postID int64, parentID *int64,
	userID int64, content string) (*blog.Comment, error) {

	content = strings.TrimSpace(content)

	if content == "" {
		return nil, errors.New("内容不能为空")
	}
	if len(content) > 4000 {
		return nil, errors.New("评论最长 4000 个字符")
	}
	if parentID != nil {
		parent, err := s.commentRepo.GetByID(ctx, *parentID)
		if err != nil {
			return nil, errors.New("回复的评论不存在")
		}
		if parent.PostID != postID {
			return nil, errors.New("不能回复其他文章的评论")
		}
		if parent.ParentID != nil {
			return nil, errors.New("评论只支持一级回复")
		}
	}

	c := &blog.Comment{
		PostID:   postID,
		ParentID: parentID,
		UserID:   userID,
		Content:  s.policy.Sanitize(content),
	}
	if err := s.commentRepo.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// ListByPostID 取某篇文章的评论
func (s *CommentLogic) ListByPostID(ctx context.Context, postID int64) ([]blog.Comment, error) {
	return s.commentRepo.ListByPostID(ctx, postID)
}

// GetByID 取一条评论
func (s *CommentLogic) GetByID(ctx context.Context, id int64) (*blog.Comment, error) {
	return s.commentRepo.GetByID(ctx, id)
}

// Delete 删除一条评论
func (s *CommentLogic) Delete(ctx context.Context, id int64) error {
	return s.commentRepo.Delete(ctx, id)
}
