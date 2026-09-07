package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool/logic"
)

// Handler 通过 HTTP 提供工具端点。
type Handler struct {
	jsonLogic               *logic.JSONLogic
	courseRegistrationLogic *logic.CourseRegistrationLogic
}

func New(jsonLogic *logic.JSONLogic, courseRegistrationLogic *logic.CourseRegistrationLogic) *Handler {
	return &Handler{jsonLogic: jsonLogic, courseRegistrationLogic: courseRegistrationLogic}
}

func (h *Handler) RegisterRoutes(r chi.Router) {
	r.Post("/tools/json/format", h.format)
	r.With(authmid.RequireUser).Post("/tools/course-registration/login", h.courseRegistrationLogin)
	r.With(authmid.RequireUser).Post("/tools/course-registration/open", h.courseRegistrationOpen)
}

// FormatJSONRequest 是 JSON 格式化器的请求体。
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

// CourseRegistrationLoginRequest 是北京交通大学选课系统登录请求。
type CourseRegistrationLoginRequest struct {
	StudentID      string `json:"student_id"`
	Password       string `json:"password"`
	TujianUsername string `json:"tujian_username"`
	TujianPassword string `json:"tujian_password"`
}

func (r CourseRegistrationLoginRequest) Validate() error {
	if strings.TrimSpace(r.StudentID) == "" && r.Password == "" &&
		strings.TrimSpace(r.TujianUsername) == "" && r.TujianPassword == "" {
		return nil
	}
	if strings.TrimSpace(r.StudentID) == "" {
		return errors.New("student_id is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	if strings.TrimSpace(r.TujianUsername) == "" {
		return errors.New("tujian_username is required")
	}
	if r.TujianPassword == "" {
		return errors.New("tujian_password is required")
	}
	return nil
}

func (h *Handler) courseRegistrationLogin(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[CourseRegistrationLoginRequest](w, r, 16<<10)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	principal := authmid.GetPrincipal(r.Context())
	data, code, err := h.courseRegistrationLogic.Login(
		r.Context(),
		principal.ID,
		req.StudentID,
		req.Password,
		req.TujianUsername,
		req.TujianPassword,
	)
	httpx.Reply(r.Context(), w, data, code, err)
}

func (h *Handler) courseRegistrationOpen(w http.ResponseWriter, r *http.Request) {
	principal := authmid.GetPrincipal(r.Context())
	data, code, err := h.courseRegistrationLogic.OpenCourseSelection(r.Context(), principal.ID)
	httpx.Reply(r.Context(), w, data, code, err)
}
