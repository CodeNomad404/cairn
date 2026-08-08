package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/auth"
	authmid "cairn.cn/internal/auth/middleware"
	bloglogic "cairn.cn/internal/blog/logic"
	"cairn.cn/internal/httpx"
)

// PostHandler 通过 HTTP 提供博客文章端点。
type PostHandler struct {
	postLogic *bloglogic.PostLogic
}

func NewPostHandler(postLogic *bloglogic.PostLogic) *PostHandler {
	return &PostHandler{postLogic: postLogic}
}

func (h *PostHandler) RegisterRoutes(r chi.Router) {
	r.With(authmid.RequireRole(auth.RoleAdmin)).Post("/posts", h.create)
	r.With(authmid.RequireRole(auth.RoleAdmin)).Delete("/posts/{slug}", h.delete)
	r.Get("/posts", h.list)
	r.Get("/posts/{slug}", h.get)
}

// CreatePostRequest 是创建博客文章的请求体。
type CreatePostRequest struct {
	Slug      string `json:"slug"`
	Title     string `json:"title"`
	ContentMD string `json:"content_md"`
}

func (r CreatePostRequest) Validate() error {
	if strings.TrimSpace(r.Slug) == "" {
		return errors.New("slug is required")
	}
	if strings.TrimSpace(r.Title) == "" {
		return errors.New("title is required")
	}
	if strings.TrimSpace(r.ContentMD) == "" {
		return errors.New("content_md is required")
	}
	return nil
}

// create 创建一篇博客文章。
func (h *PostHandler) create(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[CreatePostRequest](w, r, 1<<20)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.postLogic.Create(r.Context(), req.Slug, req.Title, req.ContentMD)
	httpx.Reply(r.Context(), w, data, code, err)
}

// list 返回已发布的博客文章列表。
func (h *PostHandler) list(w http.ResponseWriter, r *http.Request) {
	pagination, err := httpx.ParsePagination(r)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.postLogic.List(r.Context(), pagination.Page, pagination.PageSize)
	httpx.Reply(r.Context(), w, data, code, err)
}

// get 按 slug 返回一篇博客文章。
func (h *PostHandler) get(w http.ResponseWriter, r *http.Request) {
	slug, err := httpx.ParsePath(r, "slug")
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.postLogic.Get(r.Context(), slug)
	httpx.Reply(r.Context(), w, data, code, err)
}

// delete 软删除一篇博客文章。
func (h *PostHandler) delete(w http.ResponseWriter, r *http.Request) {
	slug, err := httpx.ParsePath(r, "slug")
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	code, err := h.postLogic.Delete(r.Context(), slug)
	httpx.Reply(r.Context(), w, nil, code, err)
}
