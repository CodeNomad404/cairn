package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/auth"
	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/blog"
	"cairn.cn/internal/web"
)

type commentPostSource interface {
	GetBySlug(ctx context.Context, slug string) (*blog.Post, error)
}

type commentSource interface {
	Create(ctx context.Context, postID int64, parentID *int64, userID int64, content string) (*blog.Comment, error)
	ListByPostID(ctx context.Context, postID int64) ([]blog.Comment, error)
	GetByID(ctx context.Context, id int64) (*blog.Comment, error)
	Delete(ctx context.Context, id int64) error
}

type CommentHandler struct {
	posts    commentPostSource
	comments commentSource
}

func NewCommentHandler(posts commentPostSource, comments commentSource) *CommentHandler {
	return &CommentHandler{posts: posts, comments: comments}
}

func (h *CommentHandler) RegisterRoutes(r chi.Router) {
	r.Get("/posts/{slug}/comments", h.list)
	r.Post("/posts/{slug}/comments", h.create)
	r.Delete("/comments/{id}", h.delete)
}

func (h *CommentHandler) list(w http.ResponseWriter, r *http.Request) {
	post, ok := h.post(w, r)
	if !ok {
		return
	}
	comments, err := h.comments.ListByPostID(r.Context(), post.ID)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load comments")
		return
	}
	web.WriteJSON(w, http.StatusOK, struct {
		Items []blog.Comment `json:"items"`
	}{Items: comments})
}

func (h *CommentHandler) create(w http.ResponseWriter, r *http.Request) {
	user := authmid.GetUser(r.Context())
	if user == nil {
		web.WriteError(w, http.StatusUnauthorized, "unauthorized", "login required")
		return
	}
	post, ok := h.post(w, r)
	if !ok {
		return
	}
	var input struct {
		Content  string `json:"content"`
		ParentID *int64 `json:"parent_id"`
	}
	if !web.DecodeJSON(w, r, &input, 32<<10) {
		return
	}
	comment, err := h.comments.Create(r.Context(), post.ID, input.ParentID, user.ID, input.Content)
	if err != nil {
		web.WriteError(w, http.StatusUnprocessableEntity, "invalid_comment", err.Error())
		return
	}
	web.WriteJSON(w, http.StatusCreated, comment)
}

func (h *CommentHandler) delete(w http.ResponseWriter, r *http.Request) {
	user := authmid.GetUser(r.Context())
	if user == nil {
		web.WriteError(w, http.StatusUnauthorized, "unauthorized", "login required")
		return
	}
	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		web.WriteError(w, http.StatusBadRequest, "invalid_request", "comment id must be a positive integer")
		return
	}
	comment, err := h.comments.GetByID(r.Context(), id)
	if err != nil {
		web.WriteError(w, http.StatusNotFound, "not_found", "comment not found")
		return
	}
	if user.Role != auth.RoleAdmin && comment.UserID != user.ID {
		web.WriteError(w, http.StatusForbidden, "forbidden", "cannot delete this comment")
		return
	}
	if err := h.comments.Delete(r.Context(), id); err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to delete comment")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *CommentHandler) post(w http.ResponseWriter, r *http.Request) (*blog.Post, bool) {
	post, err := h.posts.GetBySlug(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, blog.ErrPostNotFound) {
		web.WriteError(w, http.StatusNotFound, "not_found", "post not found")
		return nil, false
	}
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load post")
		return nil, false
	}
	return post, true
}
