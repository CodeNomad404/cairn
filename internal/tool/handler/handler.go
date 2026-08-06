package handler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/tool/logic"
	"cairn.cn/internal/web"
)

type Handler struct {
	jsonSvc *logic.JSONLogic
}

func New(jsonSvc *logic.JSONLogic) *Handler {
	return &Handler{jsonSvc: jsonSvc}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/tools/json/format", h.format)
}

func (h *Handler) format(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Input string `json:"input"`
	}
	if !web.DecodeJSON(w, r, &input, 1<<20) {
		return
	}
	formatted, err := h.jsonSvc.Format(input.Input)
	if err != nil {
		web.WriteError(w, http.StatusUnprocessableEntity, "invalid_json", err.Error())
		return
	}
	web.WriteJSON(w, http.StatusOK, struct {
		Formatted string `json:"formatted"`
	}{Formatted: formatted})
}
