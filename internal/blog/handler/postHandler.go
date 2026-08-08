package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/auth"
	authmid "cairn.cn/internal/auth/middleware"
	bloglogic "cairn.cn/internal/blog/logic"
	"cairn.cn/internal/svc"
	"cairn.cn/internal/util"
)

// PostHandler exposes blog posts over HTTP.
type PostHandler struct {
	svcCtx *svc.ServiceContext
}

func NewPostHandler(svcCtx *svc.ServiceContext) *PostHandler {
	return &PostHandler{svcCtx: svcCtx}
}

func (h *PostHandler) RegisterRoutes(r chi.Router) {
	r.With(authmid.RequireRole(auth.RoleAdmin)).
		Post("/posts", util.ParseBody[CreatePostRequest](1<<20, h.create))
	r.Get("/posts", util.ParsePagination(h.list))
	r.Get("/posts/{slug}", util.ParsePath("slug", h.get))
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
func (h *PostHandler) create(w http.ResponseWriter, r *http.Request, req *CreatePostRequest) {
	l := bloglogic.NewPostLogic(r.Context(), h.svcCtx)
	data, code, msg := l.Create(req.Slug, req.Title, req.ContentMD, authmid.GetUser(r.Context()))
	util.Reply(r.Context(), w, data, code, msg)
}

// list returns published blog posts.
func (h *PostHandler) list(w http.ResponseWriter, r *http.Request, pagination util.Pagination) {
	l := bloglogic.NewPostLogic(r.Context(), h.svcCtx)
	data, code, msg := l.List(pagination.Page, pagination.PageSize)
	util.Reply(r.Context(), w, data, code, msg)
}

// get returns one blog post by slug.
func (h *PostHandler) get(w http.ResponseWriter, r *http.Request, slug string) {
	l := bloglogic.NewPostLogic(r.Context(), h.svcCtx)
	data, code, msg := l.Get(slug)
	util.Reply(r.Context(), w, data, code, msg)
}
