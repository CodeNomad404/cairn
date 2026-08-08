package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/auth"
	"cairn.cn/internal/auth/logic"
	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/httpx"
)

// add
// AuthHandler exposes auth endpoints over HTTP.
type AuthHandler struct {
	authLogic     *logic.AuthLogic
	secureCookies bool
}

func NewAuthHandler(authLogic *logic.AuthLogic, secureCookies bool) *AuthHandler {
	return &AuthHandler{authLogic: authLogic, secureCookies: secureCookies}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/register", h.register)
	r.Post("/login", h.login)
	r.Post("/logout", h.logout)
	r.With(authmid.RequireUser).Get("/me", h.me)
}

// RegisterRequest is the request body for registration.
type RegisterRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (r RegisterRequest) Validate() error {
	if strings.TrimSpace(r.Username) == "" {
		return errors.New("username is required")
	}
	if r.Password == "" {
		return errors.New("password is required")
	}
	return nil
}

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	req, err := httpx.ParseBody[RegisterRequest](w, r, 16<<10)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, httpx.CodeBadRequest, err)
		return
	}
	code, err := h.authLogic.Register(r.Context(), req.Username, req.Password)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, code, err)
		return
	}

	result, loginCode, err := h.authLogic.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		httpx.Reply(r.Context(), w, nil, loginCode, err)
		return
	}
	h.setSessionCookie(w, result.Token)
	httpx.Reply(r.Context(), w, result.User, code, nil)
}

// LoginRequest is the request body for login.
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
	user := authmid.GetUser(r.Context())
	httpx.Reply(r.Context(), w, user, httpx.CodeSuccess, nil)
}
