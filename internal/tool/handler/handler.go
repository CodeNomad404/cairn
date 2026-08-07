package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/request"
	"cairn.cn/internal/response"
	"cairn.cn/internal/svc"
	"cairn.cn/internal/tool/logic"
)

// Handler exposes tool endpoints over HTTP.
type Handler struct {
	svcCtx *svc.ServiceContext
}

func New(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{svcCtx: svcCtx}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/tools/json/format", request.WithValidatedJSON[logic.FormatJSONRequest](1<<20, h.format))
}

func (h *Handler) format(w http.ResponseWriter, r *http.Request, req *logic.FormatJSONRequest) {
	l := logic.NewJSONLogic(r.Context(), h.svcCtx)
	data, code, msg := l.Format(req.Input)
	response.Reply(r.Context(), w, data, code, msg)
}
