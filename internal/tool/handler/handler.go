package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool/logic"
)

// Handler exposes tool endpoints over HTTP.
type Handler struct {
	jsonLogic *logic.JSONLogic
}

func New(jsonLogic *logic.JSONLogic) *Handler {
	return &Handler{jsonLogic: jsonLogic}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/tools/json/format", h.format)
}

// FormatJSONRequest is the request body for the JSON formatter.
type FormatJSONRequest struct {
	Input string `json:"input"`
}

func (r FormatJSONRequest) Validate() error {
	return nil
}

func (h *Handler) format(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[FormatJSONRequest](w, r, 1<<20)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := h.jsonLogic.Format(req.Input)
	httpx.Reply(r.Context(), w, data, code, err)
}
