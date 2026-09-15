package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool/logic"
)

var classroomLocation = time.FixedZone("Asia/Shanghai", 8*60*60)

type ClassroomHandler struct {
	classroomLogic *logic.ClassroomLogic
}

func NewClassroomHandler(classroomLogic *logic.ClassroomLogic) *ClassroomHandler {
	return &ClassroomHandler{classroomLogic: classroomLogic}
}

func (h *ClassroomHandler) RegisterRoutes(r chi.Router) {
	r.With(authmid.RequireUser).Get("/tool/classroom/list", h.list)
}

func (h *ClassroomHandler) list(w http.ResponseWriter, r *http.Request) {
	date := time.Now().In(classroomLocation)
	// 解析 period 参数，范围为 1-7
	period, err := httpx.ParseQueryInt(r, "period", 0, 0, 7)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	// 解析 building_id 参数，长度为 16
	buildingID, err := httpx.ParseQueryString(r, "building_id", 16)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}

	// 解析 min_capacity 参数，范围为 0-10000
	minCapacity, err := httpx.ParseQueryInt(r, "min_capacity", 0, 0, 10000)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	principal := authmid.GetPrincipal(r.Context())
	data, code, err := h.classroomLogic.List(
		r.Context(), principal.ID, date, period, buildingID, minCapacity,
	)
	httpx.Reply(r.Context(), w, data, code, err)
}
