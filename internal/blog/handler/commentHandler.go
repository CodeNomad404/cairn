package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/auth"
	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/blog"
	"cairn.cn/internal/blog/service"
	"cairn.cn/internal/web"
)

// CommentHandler 评论相关的 HTTP 处理器
type CommentHandler struct {
	postSvc    *service.PostService
	commentSvc *service.CommentService
	r          *web.Renderer
}

// NewCommentHandler 创建 CommentHandler 实例
func NewCommentHandler(postSvc *service.PostService, commentSvc *service.CommentService, renderer *web.Renderer) *CommentHandler {
	return &CommentHandler{postSvc: postSvc, commentSvc: commentSvc, r: renderer}
}

type commentSection struct {
	Slug     string
	Comments []blog.Comment
	User     *auth.User
}

// RegisterRoutes 向 chi 路由注册评论相关路由
func (h *CommentHandler) RegisterRoutes(r chi.Router) {
	r.With(web.RateLimit(10, time.Minute)).Post("/posts/{slug}/comments", h.create)
	r.Delete("/comments/{id}", h.delete)
}

// create 处理评论提交
func (h *CommentHandler) create(w http.ResponseWriter, r *http.Request) {
	user := authmid.GetUser(r.Context())
	if user == nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	slug := slugParam(r)
	post, err := h.postSvc.GetBySlug(r.Context(), slug)
	if err != nil {
		http.NotFound(w, r)
		return
	}

	parentIDStr := r.FormValue("parent_id")
	var parentID *int64
	if pid, err := strconv.ParseInt(parentIDStr, 10, 64); err == nil && pid > 0 {
		parentID = &pid
	}

	comment, err := h.commentSvc.Create(r.Context(), post.ID, parentID,
		user.ID,
		r.FormValue("content"),
	)

	isHX := r.Header.Get("HX-Request") == "true"

	if err != nil {
		if isHX {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			w.WriteHeader(http.StatusUnprocessableEntity)
			_, _ = w.Write([]byte(`<div class="text-red-600 text-sm mt-2">` + err.Error() + `</div>`))
			return
		}
		http.Error(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	if isHX {
		comments, err := h.commentSvc.ListByPostID(r.Context(), post.ID)
		if err != nil {
			http.Error(w, "failed to load comments", http.StatusInternalServerError)
			return
		}
		h.r.Render(w, "comment-section", commentSection{
			Slug:     post.Slug,
			Comments: comments,
			User:     user,
		})
		return
	}

	http.Redirect(w, r, "/posts/"+slug+"#comments", http.StatusSeeOther)
	_ = comment
}

// delete 处理评论删除
func (h *CommentHandler) delete(w http.ResponseWriter, r *http.Request) {
	user := authmid.GetUser(r.Context())
	if user == nil {
		http.Error(w, "login required", http.StatusUnauthorized)
		return
	}

	idStr := chi.URLParam(r, "id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		http.Error(w, "invalid id", http.StatusBadRequest)
		return
	}

	comment, err := h.commentSvc.GetByID(r.Context(), id)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	if user.Role != "admin" && comment.UserID != user.ID {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}

	if err := h.commentSvc.Delete(r.Context(), id); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}
