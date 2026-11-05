package handler

import (
	"errors"
	"net/http"
	"strings"

	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool/logic"
	"github.com/go-chi/chi/v5"
)

// RegisterMISBind exposes one shared endpoint without introducing another module.
func RegisterMISBind(r chi.Router, l *logic.CourseRegistrationLogic) {
	r.With(authmid.RequireUser).Post("/tool/mis-account/bind", func(w http.ResponseWriter, r *http.Request) { bindMIS(w, r, l) })
}

type MISBindRequest struct {
	StudentID string `json:"student_id"`
	Password  string `json:"password"`
}

func (r MISBindRequest) Validate() error {
	if strings.TrimSpace(r.StudentID) == "" || len(r.StudentID) > 64 {
		return errors.New("学号不能为空且不能超过 64 字节")
	}
	if r.Password == "" || len(r.Password) > 256 {
		return errors.New("密码不能为空且不能超过 256 字节")
	}
	return nil
}

func bindMIS(w http.ResponseWriter, r *http.Request, l *logic.CourseRegistrationLogic) {
	req, err := httpx.ParseBody[MISBindRequest](w, r, 4096)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	data, code, err := l.BindMIS(r.Context(), authmid.GetPrincipal(r.Context()).ID, req.StudentID, req.Password)
	httpx.Reply(r.Context(), w, data, code, err)
}
