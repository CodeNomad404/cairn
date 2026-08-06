package handler

import (
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/auth"
	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/blog"
	bloglogic "cairn.cn/internal/blog/logic"
	"cairn.cn/internal/web"
)

const (
	defaultPageSize = 20
	maxPageSize     = 100
)

type postSource interface {
	GetBySlug(ctx context.Context, slug string) (*blog.Post, error)
	ListPublished(ctx context.Context, limit, offset int) ([]blog.Post, error)
	Publish(ctx context.Context, req *bloglogic.PublishPostRequest) (*blog.Post, error)
}

// PostHandler exposes blog posts over HTTP.
type PostHandler struct {
	posts postSource
}

func NewPostHandler(posts postSource) *PostHandler {
	return &PostHandler{posts: posts}
}

func (h *PostHandler) RegisterRoutes(r chi.Router) {
	r.Get("/posts", h.list)
	r.Get("/posts/{slug}", h.detail)
	r.With(authmid.RequireRole(auth.RoleAdmin)).Post("/posts", h.create)
}

type postSummary struct {
	ID          int64      `json:"id"`
	Slug        string     `json:"slug"`
	Title       string     `json:"title"`
	PublishedAt *time.Time `json:"published_at"`
}

type postDetail struct {
	postSummary
	ContentHTML string `json:"content_html"`
}

type postListResponse struct {
	Items    []postSummary `json:"items"`
	Page     int           `json:"page"`
	PageSize int           `json:"page_size"`
	HasMore  bool          `json:"has_more"`
}

/**
* 文章列表
 */
func (h *PostHandler) list(w http.ResponseWriter, r *http.Request) {
	page, pageSize, ok := pagination(w, r)
	if !ok {
		return
	}

	posts, err := h.posts.ListPublished(r.Context(), pageSize+1, (page-1)*pageSize)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load posts")
		return
	}

	hasMore := len(posts) > pageSize
	if hasMore {
		posts = posts[:pageSize]
	}

	items := make([]postSummary, len(posts))
	for i, post := range posts {
		items[i] = summary(post)
	}

	web.WriteJSON(w, http.StatusOK, postListResponse{
		Items:    items,
		Page:     page,
		PageSize: pageSize,
		HasMore:  hasMore,
	})
}

func (h *PostHandler) detail(w http.ResponseWriter, r *http.Request) {
	post, err := h.posts.GetBySlug(r.Context(), chi.URLParam(r, "slug"))
	if errors.Is(err, blog.ErrPostNotFound) {
		web.WriteError(w, http.StatusNotFound, "not_found", "post not found")
		return
	}
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to load post")
		return
	}

	web.WriteJSON(w, http.StatusOK, postDetail{
		postSummary: summary(*post),
		ContentHTML: post.ContentHTML,
	})
}

func (h *PostHandler) create(w http.ResponseWriter, r *http.Request) {
	var req bloglogic.PublishPostRequest
	if !web.DecodeJSON(w, r, &req, 1<<20) || !web.Validate(w, &req) {
		return
	}

	post, err := h.posts.Publish(r.Context(), &req)
	if err != nil {
		web.WriteError(w, http.StatusUnprocessableEntity, "invalid_post", err.Error())
		return
	}
	web.WriteJSON(w, http.StatusCreated, postDetail{postSummary: summary(*post), ContentHTML: post.ContentHTML})
}

func pagination(w http.ResponseWriter, r *http.Request) (int, int, bool) {
	page, err := positiveInt(r.URL.Query().Get("page"), 1)
	if err != nil {
		web.WriteError(w, http.StatusBadRequest, "invalid_request", "page must be a positive integer")
		return 0, 0, false
	}
	pageSize, err := positiveInt(r.URL.Query().Get("page_size"), defaultPageSize)
	if err != nil || pageSize > maxPageSize {
		web.WriteError(w, http.StatusBadRequest, "invalid_request", "page_size must be between 1 and 100")
		return 0, 0, false
	}
	return page, pageSize, true
}

func positiveInt(raw string, fallback int) (int, error) {
	if raw == "" {
		return fallback, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < 1 {
		return 0, errors.New("must be positive")
	}
	return value, nil
}

func summary(post blog.Post) postSummary {
	return postSummary{
		ID:          post.ID,
		Slug:        post.Slug,
		Title:       post.Title,
		PublishedAt: post.PublishedAt,
	}
}
