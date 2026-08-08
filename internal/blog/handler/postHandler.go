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

// PostHandler exposes blog posts over HTTP.
type PostHandler struct {
	postLogic *bloglogic.PostLogic
}

func NewPostHandler(postLogic *bloglogic.PostLogic) *PostHandler {
	return &PostHandler{postLogic: postLogic}
}

func (h *PostHandler) RegisterRoutes(r chi.Router) {
	r.With(authmid.RequireRole(auth.RoleAdmin)).Post("/posts", h.create)
	r.Get("/posts", h.list)
	r.Get("/posts/{slug}", h.get)
}

// CreatePostRequest is the request body for creating a blog post.
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

// create adds a blog post.
func (h *PostHandler) create(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[CreatePostRequest](w, r, 1<<20)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.postLogic.Create(r.Context(), req.Slug, req.Title, req.ContentMD)
	httpx.Reply(r.Context(), w, data, code, err)
}

// list returns published blog posts.
func (h *PostHandler) list(w http.ResponseWriter, r *http.Request) {
	pagination, err := httpx.ParsePagination(r)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.postLogic.List(r.Context(), pagination.Page, pagination.PageSize)
	httpx.Reply(r.Context(), w, data, code, err)
}

// get returns one blog post by slug.
func (h *PostHandler) get(w http.ResponseWriter, r *http.Request) {
	slug, err := httpx.ParsePath(r, "slug")
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.postLogic.Get(r.Context(), slug)
	httpx.Reply(r.Context(), w, data, code, err)
}
