package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/auth"
	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/blog"
	bloglogic "cairn.cn/internal/blog/logic"
	"cairn.cn/internal/request"
	"cairn.cn/internal/response"
	"cairn.cn/internal/svc"
)

// ListPostsRequest holds query parameters for listing posts.
type ListPostsRequest struct {
	Page     int
	PageSize int
}

// ParseListPosts extracts and validates pagination query parameters.
// Missing values default to page=1, page_size=20.
func ParseListPosts(r *http.Request) (ListPostsRequest, bool) {
	var req ListPostsRequest

	rawPage := r.URL.Query().Get("page")
	if rawPage == "" {
		req.Page = 1
	} else {
		n, err := strconv.Atoi(rawPage)
		if err != nil || n < 1 {
			return req, false
		}
		req.Page = n
	}

	rawSize := r.URL.Query().Get("page_size")
	if rawSize == "" {
		req.PageSize = 20
	} else {
		n, err := strconv.Atoi(rawSize)
		if err != nil || n < 1 || n > 100 {
			return req, false
		}
		req.PageSize = n
	}

	return req, true
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

func summary(post blog.Post) postSummary {
	return postSummary{
		ID:          post.ID,
		Slug:        post.Slug,
		Title:       post.Title,
		PublishedAt: post.PublishedAt,
	}
}

// PostHandler exposes blog posts over HTTP.
type PostHandler struct {
	svcCtx *svc.ServiceContext
}

func NewPostHandler(svcCtx *svc.ServiceContext) *PostHandler {
	return &PostHandler{svcCtx: svcCtx}
}

func (h *PostHandler) RegisterRoutes(r chi.Router) {
	r.Get("/posts", h.list)
	r.Get("/posts/{slug}", h.detail)
	r.With(authmid.RequireRole(auth.RoleAdmin)).
		Post("/posts", request.WithValidatedJSON[bloglogic.PublishPostRequest](1<<20, h.create))
}

/**
* 文章列表
 */
func (h *PostHandler) list(w http.ResponseWriter, r *http.Request) {
	req, ok := ParseListPosts(r)
	if !ok {
		response.Reply(r.Context(), w, nil, response.CodeBadRequest, "参数错误")
		return
	}

	l := bloglogic.NewPostLogic(r.Context(), h.svcCtx)
	data, code, msg := l.ListPublished(req.Page, req.PageSize)
	response.Reply(r.Context(), w, data, code, msg)
}

func (h *PostHandler) detail(w http.ResponseWriter, r *http.Request) {
	l := bloglogic.NewPostLogic(r.Context(), h.svcCtx)
	data, code, msg := l.GetBySlug(chi.URLParam(r, "slug"))
	if data != nil {
		post := data.(*blog.Post)
		data = postDetail{postSummary: summary(*post), ContentHTML: post.ContentHTML}
	}
	response.Reply(r.Context(), w, data, code, msg)
}

func (h *PostHandler) create(w http.ResponseWriter, r *http.Request, req *bloglogic.PublishPostRequest) {
	l := bloglogic.NewPostLogic(r.Context(), h.svcCtx)
	data, code, msg := l.Publish(req, authmid.GetUser(r.Context()))
	if data != nil {
		post := data.(*blog.Post)
		data = postDetail{postSummary: summary(*post), ContentHTML: post.ContentHTML}
	}
	response.Reply(r.Context(), w, data, code, msg)
}
