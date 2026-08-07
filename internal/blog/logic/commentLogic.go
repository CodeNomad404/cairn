package logic

import (
	"context"
	"errors"
	"strings"

	"github.com/microcosm-cc/bluemonday"

	"cairn.cn/internal/blog"
	"cairn.cn/internal/response"
	"cairn.cn/internal/svc"
)

// CreateCommentRequest is the request body for creating a comment.
type CreateCommentRequest struct {
	Content  string `json:"content"`
	ParentID *int64 `json:"parent_id"`
}

// Validate checks the API contract for creating a comment.
func (r CreateCommentRequest) Validate() error {
	if strings.TrimSpace(r.Content) == "" {
		return errors.New("content is required")
	}
	return nil
}

// CommentLogic 评论相关的业务逻辑
type CommentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	policy *bluemonday.Policy
}

// NewCommentLogic 创建 CommentLogic 实例
func NewCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CommentLogic {
	policy := bluemonday.StrictPolicy()
	return &CommentLogic{ctx: ctx, svcCtx: svcCtx, policy: policy}
}

// Create 提交一条评论
func (l *CommentLogic) Create(postID int64, parentID *int64, userID int64, content string) (any, response.Code, string) {
	content = strings.TrimSpace(content)

	if content == "" {
		return nil, response.CodeBadRequest, "content is required"
	}
	if len(content) > 4000 {
		return nil, response.CodeBadRequest, "comment cannot exceed 4000 characters"
	}
	if parentID != nil {
		parent, err := l.svcCtx.CommentRepo.GetByID(l.ctx, *parentID)
		if err != nil {
			return nil, response.CodeValidationFailed, "parent comment not found"
		}
		if parent.PostID != postID {
			return nil, response.CodeValidationFailed, "cannot reply to a comment from a different post"
		}
		if parent.ParentID != nil {
			return nil, response.CodeValidationFailed, "only one level of replies is supported"
		}
	}

	c := &blog.Comment{
		PostID:   postID,
		ParentID: parentID,
		UserID:   userID,
		Content:  l.policy.Sanitize(content),
	}
	if err := l.svcCtx.CommentRepo.Create(l.ctx, c); err != nil {
		return nil, response.CodeInternalError, "failed to create comment"
	}
	return c, response.CodeCreated, ""
}

// ListByPostID 取某篇文章的评论
func (l *CommentLogic) ListByPostID(postID int64) (any, response.Code, string) {
	comments, err := l.svcCtx.CommentRepo.ListByPostID(l.ctx, postID)
	if err != nil {
		return nil, response.CodeInternalError, "failed to load comments"
	}
	return map[string]any{"items": comments}, response.CodeSuccess, ""
}

// GetByID 取一条评论（返回 *blog.Comment 供调用方检查内部字段）。
func (l *CommentLogic) GetByID(id int64) (any, response.Code, string) {
	comment, err := l.svcCtx.CommentRepo.GetByID(l.ctx, id)
	if err != nil {
		return nil, response.CodeNotFound, "comment not found"
	}
	return comment, response.CodeSuccess, ""
}

// Delete 删除一条评论
func (l *CommentLogic) Delete(id int64) (any, response.Code, string) {
	if err := l.svcCtx.CommentRepo.Delete(l.ctx, id); err != nil {
		return nil, response.CodeInternalError, "failed to delete comment"
	}
	return nil, response.CodeNoContent, ""
}
