package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"

	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool/logic"
)

// CourseHandler 通过 HTTP 提供北京交通大学选课系统端点。
type CourseHandler struct {
	courseRegistrationLogic *logic.CourseRegistrationLogic
}

func NewCourseHandler(courseRegistrationLogic *logic.CourseRegistrationLogic) *CourseHandler {
	return &CourseHandler{courseRegistrationLogic: courseRegistrationLogic}
}

func (h *CourseHandler) RegisterRoutes(r chi.Router) {
	r.With(authmid.RequireUser).Post("/tools/course-registration/login", h.login)
	r.With(authmid.RequireUser).Get("/tools/course-registration/courses", h.listCourses)
	r.With(authmid.RequireUser).Post("/tools/course-registration/grab", h.grabCourses)
	r.With(authmid.RequireUser).Get("/tools/course-registration/grab-tasks", h.listGrabTasks)
	r.With(authmid.RequireUser).Patch("/tools/course-registration/grab-tasks/{taskID}", h.setGrabTaskStatus)
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

type CourseGrabRequest struct {
	Courses []string `json:"courses"`
}

func (r CourseGrabRequest) Validate() error {
	if len(r.Courses) == 0 {
		return errors.New("courses is required")
	}
	if len(r.Courses) > 50 {
		return errors.New("courses must contain at most 50 items")
	}
	for _, course := range r.Courses {
		if strings.TrimSpace(course) == "" {
			return errors.New("courses cannot contain empty items")
		}
		if len(course) > 256 {
			return errors.New("course keyword is too long")
		}
	}
	return nil
}

// login 处理北京交通大学选课系统登录请求。
func (h *CourseHandler) login(w http.ResponseWriter, r *http.Request) {
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

// listCourses 处理列出课程请求。
func (h *CourseHandler) listCourses(w http.ResponseWriter, r *http.Request) {
	principal := authmid.GetPrincipal(r.Context())
	data, code, err := h.courseRegistrationLogic.ListCourses(r.Context(), principal.ID)
	httpx.Reply(r.Context(), w, data, code, err)
}

// grabCourses 处理抢课请求。
func (h *CourseHandler) grabCourses(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[CourseGrabRequest](w, r, 16<<10)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	principal := authmid.GetPrincipal(r.Context())
	data, code, err := h.courseRegistrationLogic.GrabCourses(
		r.Context(), principal.ID, req.Courses,
	)
	httpx.Reply(r.Context(), w, data, code, err)
}

// listGrabTasks 处理列出抢课任务请求。
func (h *CourseHandler) listGrabTasks(w http.ResponseWriter, r *http.Request) {
	principal := authmid.GetPrincipal(r.Context())
	data, code, err := h.courseRegistrationLogic.ListCourseGrabTasks(r.Context(), principal.ID)
	httpx.Reply(r.Context(), w, data, code, err)
}

type CourseGrabTaskStatusRequest struct {
	Status string `json:"status"`
}

func (r CourseGrabTaskStatusRequest) Validate() error {
	if r.Status != "grabbing" && r.Status != "paused" {
		return errors.New("status must be grabbing or paused")
	}
	return nil
}

// setGrabTaskStatus 处理设置抢课任务状态请求。
func (h *CourseHandler) setGrabTaskStatus(w http.ResponseWriter, r *http.Request) {
	taskID, err := strconv.ParseInt(chi.URLParam(r, "taskID"), 10, 64)
	if err != nil || taskID <= 0 {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest,
			httpx.NewAppError("抢课任务 ID 无效", nil))
		return
	}
	req, err := httpx.ParseBody[CourseGrabTaskStatusRequest](w, r, 16<<10)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	principal := authmid.GetPrincipal(r.Context())
	data, code, err := h.courseRegistrationLogic.SetCourseGrabTaskStatus(
		r.Context(), principal.ID, taskID, req.Status,
	)
	httpx.Reply(r.Context(), w, data, code, err)
}
