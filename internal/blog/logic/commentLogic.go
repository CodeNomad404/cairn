package logic

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/microcosm-cc/bluemonday"

	"cairn.cn/internal/auth"
	"cairn.cn/internal/blog"
	"cairn.cn/internal/util"
)

// CommentLogic 评论相关的业务逻辑
type CommentLogic struct {
	postRepo    blog.PostRepo
	commentRepo blog.CommentRepo
	policy      *bluemonday.Policy
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

// NewCommentLogic 创建 CommentLogic 实例。
// Bluemonday Policy 初始化后只读，可安全并发复用，故只在启动时创建一次。
func NewCommentLogic(postRepo blog.PostRepo, commentRepo blog.CommentRepo) *CommentLogic {
	policy := bluemonday.StrictPolicy()
	return &CommentLogic{postRepo: postRepo, commentRepo: commentRepo, policy: policy}
}

// Create 提交一条评论
func (l *CommentLogic) Create(ctx context.Context, slug string, parentID *int64, user *auth.User, content string) (CommentResult, util.Code, error) {
	content = strings.TrimSpace(content)

	if user == nil {
		return CommentResult{}, util.CodeUnauthorized, util.NewAppError("login required", nil)
	}
	if content == "" {
		return CommentResult{}, util.CodeBadRequest, util.NewAppError("content is required", nil)
	}
	if len(content) > 4000 {
		return CommentResult{}, util.CodeBadRequest, util.NewAppError("comment cannot exceed 4000 characters", nil)
	}
	post, err := l.postRepo.GetBySlug(ctx, slug)
	if errors.Is(err, blog.ErrPostNotFound) {
		return CommentResult{}, util.CodeNotFound, util.NewAppError("post not found", err)
	}
	if err != nil {
		return CommentResult{}, util.CodeInternalError, util.NewAppError("failed to load post", err)
	}
	if parentID != nil {
		parent, err := l.commentRepo.GetByID(ctx, *parentID)
		if err != nil {
			return CommentResult{}, util.CodeValidationFailed, util.NewAppError("parent comment not found", err)
		}
		if parent.PostID != post.ID {
			return CommentResult{}, util.CodeValidationFailed, util.NewAppError("cannot reply to a comment from a different post", nil)
		}
		if parent.ParentID != nil {
			return CommentResult{}, util.CodeValidationFailed, util.NewAppError("only one level of replies is supported", nil)
		}
	}

	c := &blog.Comment{
		PostID:   post.ID,
		ParentID: parentID,
		UserID:   user.ID,
		Content:  l.policy.Sanitize(content),
	}
	if err := l.commentRepo.Create(ctx, c); err != nil {
		return CommentResult{}, util.CodeInternalError, util.NewAppError("failed to create comment", err)
	}
	c.Username = user.Username
	return toCommentResult(*c), util.CodeCreated, nil
}

// List 取某篇文章的评论
func (l *CommentLogic) List(ctx context.Context, slug string) (CommentListResult, util.Code, error) {
	post, err := l.postRepo.GetBySlug(ctx, slug)
	if errors.Is(err, blog.ErrPostNotFound) {
		return CommentListResult{}, util.CodeNotFound, util.NewAppError("post not found", err)
	}
	if err != nil {
		return CommentListResult{}, util.CodeInternalError, util.NewAppError("failed to load post", err)
	}
	comments, err := l.commentRepo.ListByPostID(ctx, post.ID)
	if err != nil {
		return CommentListResult{}, util.CodeInternalError, util.NewAppError("failed to load comments", err)
	}
	items := make([]CommentResult, len(comments))
	for i, comment := range comments {
		items[i] = toCommentResult(comment)
	}
	return CommentListResult{Items: items}, util.CodeSuccess, nil
}

// Delete 删除一条评论
func (l *CommentLogic) Delete(ctx context.Context, id int64, user *auth.User) (util.Code, error) {
	comment, err := l.commentRepo.GetByID(ctx, id)
	if err != nil {
		return util.CodeNotFound, util.NewAppError("comment not found", err)
	}
	if user == nil {
		return util.CodeUnauthorized, util.NewAppError("login required", nil)
	}
	if user.Role != auth.RoleAdmin && comment.UserID != user.ID {
		return util.CodeForbidden, util.NewAppError("cannot delete this comment", nil)
	}
	if err := l.commentRepo.Delete(ctx, id); err != nil {
		return util.CodeInternalError, util.NewAppError("failed to delete comment", err)
	}
	return util.CodeNoContent, nil
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
