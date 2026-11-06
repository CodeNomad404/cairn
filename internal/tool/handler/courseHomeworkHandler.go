package handler

import (
	"net/http"

	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool/logic"
	"github.com/go-chi/chi/v5"
)

type HomeworkHandler struct{ logic *logic.HomeworkLogic }

func NewHomeworkHandler(l *logic.HomeworkLogic) *HomeworkHandler { return &HomeworkHandler{logic: l} }

func (h *HomeworkHandler) RegisterRoutes(r chi.Router) {
	r.With(authmid.RequireUser).Get("/tool/course-homework/list", h.list)
	r.With(authmid.RequireUser).Get("/tool/course-homework/get", h.get)
	r.With(authmid.RequireUser).Post("/tool/course-homework/sync", h.sync)
}

func (h *HomeworkHandler) list(w http.ResponseWriter, r *http.Request) {
	p, err := httpx.ParsePagination(r)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	course, err := httpx.ParseQueryString(r, "course_id", 64)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	status, err := httpx.ParseQueryString(r, "status", 16)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	if status == "" {
		status = "todo"
	}
	switch status {
	case "todo", "pending", "overdue", "submitted", "upcoming", "unknown", "all":
	default:
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, httpx.NewAppError("无效的作业状态", nil))
		return
	}
	data, code, err := h.logic.List(r.Context(), authmid.GetPrincipal(r.Context()).ID, course, status, p.Page, p.PageSize)
	httpx.Reply(r.Context(), w, data, code, err)
}

func (h *HomeworkHandler) get(w http.ResponseWriter, r *http.Request) {
	id, err := httpx.ParsePositiveInt64Query(r, "id")
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.logic.Get(r.Context(), authmid.GetPrincipal(r.Context()).ID, id)
	httpx.Reply(r.Context(), w, data, code, err)
}

type HomeworkSyncRequest struct{}

func (HomeworkSyncRequest) Validate() error { return nil }

func (h *HomeworkHandler) sync(w http.ResponseWriter, r *http.Request) {
	if _, err := httpx.ParseBody[HomeworkSyncRequest](w, r, 1024); err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.logic.Synchronize(r.Context(), authmid.GetPrincipal(r.Context()).ID)
	httpx.Reply(r.Context(), w, data, code, err)
}
