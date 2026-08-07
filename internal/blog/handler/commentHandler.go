package handler

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/auth"
	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/blog"
	bloglogic "cairn.cn/internal/blog/logic"
	"cairn.cn/internal/request"
	"cairn.cn/internal/response"
	"cairn.cn/internal/svc"
)

// CommentHandler exposes comment endpoints over HTTP.
type CommentHandler struct {
	svcCtx *svc.ServiceContext
}

func NewCommentHandler(svcCtx *svc.ServiceContext) *CommentHandler {
	return &CommentHandler{svcCtx: svcCtx}
}

func (h *CommentHandler) RegisterRoutes(r chi.Router) {
	r.Get("/posts/{slug}/comments", h.list)
	r.With(authmid.RequireUser).
		Post("/posts/{slug}/comments", request.WithValidatedJSON[bloglogic.CreateCommentRequest](32<<10, h.create))
	r.With(authmid.RequireUser).Delete("/comments/{id}", h.delete)
}

func (h *CommentHandler) list(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")
	pl := bloglogic.NewPostLogic(r.Context(), h.svcCtx)
	data, code, msg := pl.GetBySlug(slug)
	if code != response.CodeSuccess {
		response.Reply(r.Context(), w, nil, code, msg)
		return
	}
	post := data.(*blog.Post)

	cl := bloglogic.NewCommentLogic(r.Context(), h.svcCtx)
	data, code, msg = cl.ListByPostID(post.ID)
	response.Reply(r.Context(), w, data, code, msg)
}

func (h *CommentHandler) create(w http.ResponseWriter, r *http.Request, req *bloglogic.CreateCommentRequest) {
	user := authmid.GetUser(r.Context())

	slug := chi.URLParam(r, "slug")
	pl := bloglogic.NewPostLogic(r.Context(), h.svcCtx)
	data, code, msg := pl.GetBySlug(slug)
	if code != response.CodeSuccess {
		response.Reply(r.Context(), w, nil, code, msg)
		return
	}
	post := data.(*blog.Post)

	cl := bloglogic.NewCommentLogic(r.Context(), h.svcCtx)
	data, code, msg = cl.Create(post.ID, req.ParentID, user.ID, req.Content)
	response.Reply(r.Context(), w, data, code, msg)
}

func (h *CommentHandler) delete(w http.ResponseWriter, r *http.Request) {
	user := authmid.GetUser(r.Context())

	id, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil || id < 1 {
		response.Reply(r.Context(), w, nil, response.CodeBadRequest, "comment id must be a positive integer")
		return
	}

	cl := bloglogic.NewCommentLogic(r.Context(), h.svcCtx)
	commentData, code, msg := cl.GetByID(id)
	if code != response.CodeSuccess {
		response.Reply(r.Context(), w, nil, code, msg)
		return
	}
	comment := commentData.(*blog.Comment)

	if user.Role != auth.RoleAdmin && comment.UserID != user.ID {
		response.Reply(r.Context(), w, nil, response.CodeForbidden, "cannot delete this comment")
		return
	}

	_, code, msg = cl.Delete(id)
	response.Reply(r.Context(), w, nil, code, msg)
}
