package handler

import (
	"net/http"
	"time"

	"cairn.cn/internal/auth"
	"cairn.cn/internal/auth/service"
	"cairn.cn/internal/web"
)

// AuthHandler 认证相关的 HTTP 处理器
type AuthHandler struct {
	svc           *service.AuthService
	r             *web.Renderer
	secureCookies bool
}

func New(svc *service.AuthService, renderer *web.Renderer, secureCookies bool) *AuthHandler {
	return &AuthHandler{svc: svc, r: renderer, secureCookies: secureCookies}
}

type authPage struct {
	web.PageData
	Error string
}

// LoginPage 登录页面 GET
func (h *AuthHandler) LoginPage(w http.ResponseWriter, r *http.Request) {
	h.r.Render(w, "login", authPage{
		PageData: web.PrivatePageData("登录 — Cairn"),
	})
}

// Login 处理登录 POST
func (h *AuthHandler) Login(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	token, _, err := h.svc.Login(r.Context(), username, password)
	if err != nil {
		h.r.RenderStatus(w, http.StatusUnprocessableEntity, "login", authPage{
			PageData: web.PrivatePageData("登录 — Cairn"),
			Error:    err.Error(),
		})
		return
	}

	h.setSessionCookie(w, token)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// RegisterPage 注册页面 GET
func (h *AuthHandler) RegisterPage(w http.ResponseWriter, r *http.Request) {
	h.r.Render(w, "register", authPage{
		PageData: web.PrivatePageData("注册 — Cairn"),
	})
}

// Register 处理注册 POST
func (h *AuthHandler) Register(w http.ResponseWriter, r *http.Request) {
	username := r.FormValue("username")
	password := r.FormValue("password")

	_, err := h.svc.Register(r.Context(), username, password)
	if err != nil {
		h.r.RenderStatus(w, http.StatusUnprocessableEntity, "register", authPage{
			PageData: web.PrivatePageData("注册 — Cairn"),
			Error:    err.Error(),
		})
		return
	}

	// 注册成功后自动登录
	token, _, err := h.svc.Login(r.Context(), username, password)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	h.setSessionCookie(w, token)
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// Logout 处理登出 POST
func (h *AuthHandler) Logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.SessionCookie); err == nil {
		_ = h.svc.Logout(r.Context(), cookie.Value)
	}

	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookie,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
		Expires:  time.Unix(1, 0),
	})
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string) {
	const sessionDuration = 30 * 24 * time.Hour
	http.SetCookie(w, &http.Cookie{
		Name:     auth.SessionCookie,
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.secureCookies,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int(sessionDuration.Seconds()),
		Expires:  time.Now().Add(sessionDuration),
	})
}
