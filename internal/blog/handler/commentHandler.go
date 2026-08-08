package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	authmid "cairn.cn/internal/auth/middleware"
	bloglogic "cairn.cn/internal/blog/logic"
	"cairn.cn/internal/svc"
	"cairn.cn/internal/util"
)

// CommentHandler exposes comment endpoints over HTTP.
type CommentHandler struct {
	svcCtx *svc.ServiceContext
}

func NewCommentHandler(svcCtx *svc.ServiceContext) *CommentHandler {
	return &CommentHandler{svcCtx: svcCtx}
}

func (h *CommentHandler) RegisterRoutes(r chi.Router) {
	r.With(authmid.RequireUser).
		Post("/posts/{slug}/comments", util.ParsePathAndBody[CreateCommentRequest]("slug", 32<<10, h.create))
	r.With(authmid.RequireUser).
		Delete("/comments/{id}", util.ParsePositiveInt64Path("id", h.delete))
	r.Get("/posts/{slug}/comments", util.ParsePath("slug", h.list))
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
func (h *CommentHandler) create(w http.ResponseWriter, r *http.Request, slug string, req *CreateCommentRequest) {
	l := bloglogic.NewCommentLogic(h.svcCtx)
	data, code, msg := l.Create(r.Context(), slug, req.ParentID, authmid.GetUser(r.Context()), req.Content)
	util.Reply(r.Context(), w, data, code, msg)
}

// delete removes a comment.
func (h *CommentHandler) delete(w http.ResponseWriter, r *http.Request, id int64) {
	l := bloglogic.NewCommentLogic(h.svcCtx)
	data, code, msg := l.Delete(r.Context(), id, authmid.GetUser(r.Context()))
	util.Reply(r.Context(), w, data, code, msg)
}

// list returns the comments for a blog post.
func (h *CommentHandler) list(w http.ResponseWriter, r *http.Request, slug string) {
	l := bloglogic.NewCommentLogic(h.svcCtx)
	data, code, msg := l.List(r.Context(), slug)
	util.Reply(r.Context(), w, data, code, msg)
}
