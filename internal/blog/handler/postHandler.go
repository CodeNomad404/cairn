package handler

import (
	"fmt"
	"html/template"
	"io"
	"net/http"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"

	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/blog"
	"cairn.cn/internal/blog/service"
	"cairn.cn/internal/web"
)

const maxUploadSize = 10 << 20 // 10 MB

// PostHandler 博客文章相关的 HTTP 处理器
type PostHandler struct {
	svc        *service.PostService
	commentSvc *service.CommentService
	r          *web.Renderer
}

// New 创建 PostHandler 实例
func New(svc *service.PostService, commentSvc *service.CommentService, renderer *web.Renderer) *PostHandler {
	return &PostHandler{svc: svc, commentSvc: commentSvc, r: renderer}
}

type homePage struct {
	web.PageData
}

type postListPage struct {
	web.PageData
	Posts []blog.Post
}

type uploadPage struct {
	web.PageData
	Success bool
	Slug    string
	Error   string
}

type postDetailPage struct {
	web.PageData
	Post     *blog.Post
	Content  template.HTML
	Slug     string
	Comments []blog.Comment
}

// RegisterRoutes 向 chi 路由注册文章相关的所有路由
func (h *PostHandler) RegisterRoutes(r chi.Router) {
	r.Get("/", h.home)
	r.Get("/posts", h.list)
	r.Get("/posts/{slug}", h.detail)
	r.Get("/upload", h.uploadPage)
	r.Post("/upload", h.upload)
}

// home 欢迎页
func (h *PostHandler) home(w http.ResponseWriter, r *http.Request) {
	h.r.Render(w, "home", homePage{
		PageData: web.PageData{
			Title:     "Cairn · 石堆碑",
			BodyClass: "welcome-page flex flex-col",
		},
	})
}

// list 文章列表页
func (h *PostHandler) list(w http.ResponseWriter, r *http.Request) {
	posts, err := h.svc.ListPublished(r.Context(), 20, 0)
	if err != nil {
		http.Error(w, "failed to load posts", http.StatusInternalServerError)
		return
	}

	h.r.Render(w, "index", postListPage{
		PageData: web.PageData{
			Title:  "博客 — Cairn",
			Active: "blog",
			User:   authmid.GetUser(r.Context()),
		},
		Posts: posts,
	})
}

// uploadPage 上传页面（GET），仅 admin
func (h *PostHandler) uploadPage(w http.ResponseWriter, r *http.Request) {
	user := authmid.GetUser(r.Context())
	if user == nil || user.Role != "admin" {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("只有站长可以发布文章"))
		return
	}
	h.r.Render(w, "upload", uploadPage{
		PageData: web.PageData{
			Title:   "上传文章 — Cairn",
			Active:  "blog",
			User:    user,
			Scripts: []string{"/static/js/admin/upload.js"},
		},
	})
}

// upload 处理文件上传（POST），仅 admin
func (h *PostHandler) upload(w http.ResponseWriter, r *http.Request) {
	user := authmid.GetUser(r.Context())
	if user == nil || user.Role != "admin" {
		w.WriteHeader(http.StatusForbidden)
		_, _ = w.Write([]byte("只有站长可以发布文章"))
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, maxUploadSize)
	if err := r.ParseMultipartForm(maxUploadSize); err != nil {
		h.renderUploadError(w, r, "文件太大，最大 10MB")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		h.renderUploadError(w, r, "请选择一个 .md 文件")
		return
	}
	defer func() {
		_ = file.Close()
	}()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext != ".md" && ext != ".markdown" {
		h.renderUploadError(w, r, "只支持 .md 或 .markdown 文件")
		return
	}

	mdBytes, err := io.ReadAll(file)
	if err != nil {
		h.renderUploadError(w, r, "读取文件失败")
		return
	}

	slug := slugFromFilename(header.Filename)
	title := extractTitle(string(mdBytes))
	if title == "" {
		title = slug
	}

	post, err := h.svc.Publish(r.Context(), slug, title, string(mdBytes))
	if err != nil {
		h.renderUploadError(w, r, fmt.Sprintf("发布失败：%v", err))
		return
	}

	h.r.Render(w, "upload", uploadPage{
		PageData: web.PageData{
			Title:   "上传文章 — Cairn",
			Active:  "blog",
			User:    authmid.GetUser(r.Context()),
			Scripts: []string{"/static/js/admin/upload.js"},
		},
		Success: true,
		Slug:    post.Slug,
	})
}

// renderUploadError 渲染带错误信息的上传页面
func (h *PostHandler) renderUploadError(w http.ResponseWriter, r *http.Request, msg string) {
	h.r.RenderStatus(w, http.StatusUnprocessableEntity, "upload", uploadPage{
		PageData: web.PageData{
			Title:   "上传文章 — Cairn",
			Active:  "blog",
			User:    authmid.GetUser(r.Context()),
			Scripts: []string{"/static/js/admin/upload.js"},
		},
		Error: msg,
	})
}

// slugFromFilename 从文件名提取 slug（去掉扩展名，转小写）
func slugFromFilename(name string) string {
	base := filepath.Base(name)
	ext := filepath.Ext(base)
	slug := strings.TrimSuffix(base, ext)
	return strings.ToLower(strings.TrimSpace(slug))
}

// extractTitle 从 Markdown 内容中提取第一个 # 标题
func extractTitle(md string) string {
	for _, line := range strings.Split(md, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(trimmed, "# "))
		}
	}
	return ""
}

// detail 文章详情页（含评论区）
func (h *PostHandler) detail(w http.ResponseWriter, r *http.Request) {
	slug := slugParam(r)
	post, err := h.svc.GetBySlug(r.Context(), slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	comments, err := h.commentSvc.ListByPostID(r.Context(), post.ID)
	if err != nil {
		http.Error(w, "failed to load comments", http.StatusInternalServerError)
		return
	}

	h.r.Render(w, "post", postDetailPage{
		PageData: web.PageData{
			Title:   post.Title + " — Cairn",
			Active:  "blog",
			Page:    "post",
			User:    authmid.GetUser(r.Context()),
			Scripts: []string{"/static/js/vendor/htmx-2.0.4.min.js", "/static/js/blog/comments.js"},
		},
		Post:     post,
		Content:  template.HTML(post.ContentHTML),
		Slug:     post.Slug,
		Comments: comments,
	})
}
