package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/tool/service"
	"cairn.cn/internal/web"
)

type Handler struct {
	jsonSvc *service.JSONService
}

func New(jsonSvc *service.JSONService) *Handler {
	return &Handler{jsonSvc: jsonSvc}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/tools/json/format", h.format)
}

func (h *Handler) format(w http.ResponseWriter, r *http.Request) {
	var input struct {
		Input string `json:"input"`
	}
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<20)).Decode(&input); err != nil {
		web.WriteError(w, http.StatusBadRequest, "invalid_request", "valid JSON is required")
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
