package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/tool/logic"
	"cairn.cn/internal/util"
)

// Handler exposes tool endpoints over HTTP.
type Handler struct {
	jsonLogic *logic.JSONLogic
}

func New(jsonLogic *logic.JSONLogic) *Handler {
	return &Handler{jsonLogic: jsonLogic}
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
	data, code, msg := h.jsonLogic.Format(req.Input)
	util.Reply(r.Context(), w, data, code, msg)
}
