package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	authmid "cairn.cn/internal/auth/middleware"
	bloglogic "cairn.cn/internal/blog/logic"
	"cairn.cn/internal/httpx"
)

// CommentHandler 通过 HTTP 提供评论端点。
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

// CreateCommentRequest 是创建评论的请求体。
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

// create 向博客文章添加一条评论。
func (h *CommentHandler) create(w http.ResponseWriter, r *http.Request) {
	slug, err := httpx.ParsePath(r, "slug")
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	req, err := httpx.ParseBody[CreateCommentRequest](w, r, 32<<10)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.commentLogic.Create(r.Context(), slug, req.ParentID, authmid.GetUser(r.Context()), req.Content)
	httpx.Reply(r.Context(), w, data, code, err)
}

// delete 删除一条评论。
func (h *CommentHandler) delete(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.ParsePositiveInt64Path(r, "id")
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	code, err := h.commentLogic.Delete(r.Context(), id, authmid.GetUser(r.Context()))
	httpx.Reply(r.Context(), w, nil, code, err)
}

// list 返回某篇博客文章的评论。
func (h *CommentHandler) list(w http.ResponseWriter, r *http.Request) {
	slug, err := httpx.ParsePath(r, "slug")
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.commentLogic.List(r.Context(), slug)
	httpx.Reply(r.Context(), w, data, code, err)
}
