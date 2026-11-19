package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/auth"
	"cairn.cn/internal/auth/logic"
	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/httpx"
	webmw "cairn.cn/internal/middleware"
)

// add
// AuthHandler 通过 HTTP 提供认证端点。
type AuthHandler struct {
	authLogic     *logic.AuthLogic
	secureCookies bool
}

func NewAuthHandler(authLogic *logic.AuthLogic, secureCookies bool) *AuthHandler {
	return &AuthHandler{authLogic: authLogic, secureCookies: secureCookies}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.With(webmw.RateLimit(5, time.Minute)).Post("/register", h.register)
	r.With(webmw.RateLimit(10, time.Minute)).Post("/login", h.login)
	r.Post("/logout", h.logout)
	r.With(authmid.RequireUser).Get("/me", h.me)
}

// RegisterRequest 是注册的请求体。
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r RegisterRequest) Validate() error {
	username := strings.TrimSpace(r.Username)
	if length := utf8.RuneCountInString(username); length < 2 || length > 64 {
		return errors.New("用户名需要 2–64 个字符")
	}
	if strings.ContainsFunc(r.Username, unicode.IsControl) {
		return errors.New("用户名不能包含控制字符")
	}
	if utf8.RuneCountInString(r.Password) < 8 {
		return errors.New("密码至少需要 8 个字符")
	}
	if len(r.Password) > 72 {
		return errors.New("密码不能超过 72 字节，请缩短密码")
	}
	return nil
}

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[RegisterRequest](w, r, 16<<10)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	result, code, err := h.authLogic.Register(r.Context(), req.Username, req.Password)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, code, err)
		return
	}

	h.setSessionCookie(w, result.Token)
	httpx.Reply(r.Context(), w, result.User, code, nil)
}

// LoginRequest 是登录的请求体。
type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r LoginRequest) Validate() error {
	if strings.TrimSpace(r.Username) == "" {
		return errors.New("username is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[LoginRequest](w, r, 16<<10)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	result, code, err := h.authLogic.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, code, err)
		return
	}
	h.setSessionCookie(w, result.Token)
	httpx.Reply(r.Context(), w, result.User, code, nil)
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	code := httpx.CodeNoContent
	var logoutErr error
	if cookie, err := r.Cookie(auth.SessionCookie); err == nil {
		code, logoutErr = h.authLogic.Logout(r.Context(), cookie.Value)
	}
	h.clearSessionCookie(w)
	httpx.Reply(r.Context(), w, nil, code, logoutErr)
}

func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	principal := authmid.GetPrincipal(r.Context())
	data, code, err := h.authLogic.Me(r.Context(), principal)
	httpx.Reply(r.Context(), w, data, code, err)
}
