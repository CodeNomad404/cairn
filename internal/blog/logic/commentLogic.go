package logic

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"

	"cairn.cn/internal/auth"
	"cairn.cn/internal/blog"
	"cairn.cn/internal/svc"
	"cairn.cn/internal/util"
)

// CommentLogic 评论相关的业务逻辑
type CommentLogic struct {
	ctx    context.Context
	svcCtx *svc.ServiceContext
	policy *bluemonday.Policy
}

// CommentResult is the complete API output for one comment.
type CommentResult struct {
	ID        int64           `json:"id"`
	PostID    int64           `json:"post_id"`
	ParentID  *int64          `json:"parent_id"`
	UserID    int64           `json:"user_id"`
	Content   string          `json:"content"`
	Status    string          `json:"status"`
	CreatedAt time.Time       `json:"created_at"`
	Username  string          `json:"username"`
	Replies   []CommentResult `json:"replies,omitempty"`
}

// CommentListResult is the complete API output for a comment list.
type CommentListResult struct {
	Items []CommentResult `json:"items"`
}

// NewCommentLogic 创建 CommentLogic 实例
func NewCommentLogic(ctx context.Context, svcCtx *svc.ServiceContext) *CommentLogic {
	policy := bluemonday.StrictPolicy()
	return &CommentLogic{ctx: ctx, svcCtx: svcCtx, policy: policy}
}

// Create 提交一条评论
func (l *CommentLogic) Create(slug string, parentID *int64, userID int64, content string) (any, util.Code, string) {
	content = strings.TrimSpace(content)

	if content == "" {
		return nil, util.CodeBadRequest, "content is required"
	}
	if len(content) > 4000 {
		return nil, util.CodeBadRequest, "comment cannot exceed 4000 characters"
	}
	post, err := l.svcCtx.PostRepo.GetBySlug(l.ctx, slug)
	if errors.Is(err, blog.ErrPostNotFound) {
		return nil, util.CodeNotFound, "post not found"
	}
	if err != nil {
		return nil, util.CodeInternalError, "failed to load post"
	}
	if parentID != nil {
		parent, err := l.svcCtx.CommentRepo.GetByID(l.ctx, *parentID)
		if err != nil {
			return nil, util.CodeValidationFailed, "parent comment not found"
		}
		if parent.PostID != post.ID {
			return nil, util.CodeValidationFailed, "cannot reply to a comment from a different post"
		}
		if parent.ParentID != nil {
			return nil, util.CodeValidationFailed, "only one level of replies is supported"
		}
	}

	c := &blog.Comment{
		PostID:   post.ID,
		ParentID: parentID,
		UserID:   userID,
		Content:  l.policy.Sanitize(content),
	}
	if err := l.svcCtx.CommentRepo.Create(l.ctx, c); err != nil {
		return nil, util.CodeInternalError, "failed to create comment"
	}
	return toCommentResult(*c), util.CodeCreated, ""
}

// List 取某篇文章的评论
func (l *CommentLogic) List(slug string) (any, util.Code, string) {
	post, err := l.svcCtx.PostRepo.GetBySlug(l.ctx, slug)
	if errors.Is(err, blog.ErrPostNotFound) {
		return nil, util.CodeNotFound, "post not found"
	}
	if err != nil {
		return nil, util.CodeInternalError, "failed to load post"
	}
	comments, err := l.svcCtx.CommentRepo.ListByPostID(l.ctx, post.ID)
	if err != nil {
		return nil, util.CodeInternalError, "failed to load comments"
	}
	items := make([]CommentResult, len(comments))
	for i, comment := range comments {
		items[i] = toCommentResult(comment)
	}
	return CommentListResult{Items: items}, util.CodeSuccess, ""
}

// Delete 删除一条评论
func (l *CommentLogic) Delete(id int64, user *auth.User) (any, util.Code, string) {
	comment, err := l.svcCtx.CommentRepo.GetByID(l.ctx, id)
	if err != nil {
		return nil, util.CodeNotFound, "comment not found"
	}
	if user == nil {
		return nil, util.CodeUnauthorized, "login required"
	}
	if user.Role != auth.RoleAdmin && comment.UserID != user.ID {
		return nil, util.CodeForbidden, "cannot delete this comment"
	}
	if err := l.svcCtx.CommentRepo.Delete(l.ctx, id); err != nil {
		return nil, util.CodeInternalError, "failed to delete comment"
	}
	return nil, util.CodeNoContent, ""
}

func toCommentResult(comment blog.Comment) CommentResult {
	replies := make([]CommentResult, len(comment.Replies))
	for i, reply := range comment.Replies {
		replies[i] = toCommentResult(reply)
	}
	return CommentResult{
		ID:        comment.ID,
		PostID:    comment.PostID,
		ParentID:  comment.ParentID,
		UserID:    comment.UserID,
		Content:   comment.Content,
		Status:    comment.Status,
		CreatedAt: comment.CreatedAt,
		Username:  comment.Username,
		Replies:   replies,
	}
}
