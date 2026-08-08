package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/svc"
	"cairn.cn/internal/tool/logic"
	"cairn.cn/internal/util"
)

// Handler exposes tool endpoints over HTTP.
type Handler struct {
	svcCtx *svc.ServiceContext
}

func New(svcCtx *svc.ServiceContext) *Handler {
	return &Handler{svcCtx: svcCtx}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/tools/json/format", util.ParseBody[FormatJSONRequest](1<<20, h.format))
}

// FormatJSONRequest is the request body for the JSON formatter.
type FormatJSONRequest struct {
	Input string `json:"input"`
}

func (r FormatJSONRequest) Validate() error {
	return nil
}

func (h *Handler) format(w http.ResponseWriter, r *http.Request, req *FormatJSONRequest) {
	l := logic.NewJSONLogic(h.svcCtx)
	data, code, msg := l.Format(req.Input)
	util.Reply(r.Context(), w, data, code, msg)
}
