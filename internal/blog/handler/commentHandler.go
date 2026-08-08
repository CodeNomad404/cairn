package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	authmid "cairn.cn/internal/auth/middleware"
	bloglogic "cairn.cn/internal/blog/logic"
	"cairn.cn/internal/util"
)

// CommentHandler exposes comment endpoints over HTTP.
type CommentHandler struct {
	commentLogic *bloglogic.CommentLogic
}

func NewCommentHandler(commentLogic *bloglogic.CommentLogic) *CommentHandler {
	return &CommentHandler{commentLogic: commentLogic}
}

func (h *CommentHandler) RegisterRoutes(r chi.Router) {
	r.With(authmid.RequireUser).Post("/posts/{slug}/comments", h.create)
	r.With(authmid.RequireUser).Delete("/comments/{id}", h.delete)
	r.Get("/posts/{slug}/comments", h.list)
}

// CreateCommentRequest is the request body for creating a comment.
type CreateCommentRequest struct {
	Content  string `json:"content"`
	ParentID *int64 `json:"parent_id"`
}

func (r CreateCommentRequest) Validate() error {
	if strings.TrimSpace(r.Content) == "" {
		return errors.New("content is required")
	}
	return nil
}

// create adds a comment to a blog post.
func (h *CommentHandler) create(w http.ResponseWriter, r *http.Request) {
	slug, err := util.ParsePath(r, "slug")
	if err != nil {
		util.Reply(r.Context(), w, nil, util.CodeBadRequest, err)
		return
	}
	req, err := util.ParseBody[CreateCommentRequest](w, r, 32<<10)
	if err != nil {
		util.Reply(r.Context(), w, nil, util.CodeBadRequest, err)
		return
	}
	data, code, err := h.commentLogic.Create(r.Context(), slug, req.ParentID, authmid.GetUser(r.Context()), req.Content)
	util.Reply(r.Context(), w, data, code, err)
}

// delete removes a comment.
func (h *CommentHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := util.ParsePositiveInt64Path(r, "id")
	if err != nil {
		util.Reply(r.Context(), w, nil, util.CodeBadRequest, err)
		return
	}
	code, err := h.commentLogic.Delete(r.Context(), id, authmid.GetUser(r.Context()))
	util.Reply(r.Context(), w, nil, code, err)
}

// list returns the comments for a blog post.
func (h *CommentHandler) list(w http.ResponseWriter, r *http.Request) {
	slug, err := util.ParsePath(r, "slug")
	if err != nil {
		util.Reply(r.Context(), w, nil, util.CodeBadRequest, err)
		return
	}
	data, code, err := h.commentLogic.List(r.Context(), slug)
	util.Reply(r.Context(), w, data, code, err)
}
