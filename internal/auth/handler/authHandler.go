package handler

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/auth"
	"cairn.cn/internal/auth/logic"
	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/request"
	"cairn.cn/internal/response"
	"cairn.cn/internal/svc"
)

// AuthHandler exposes auth endpoints over HTTP.
type AuthHandler struct {
	svcCtx        *svc.ServiceContext
	secureCookies bool
}

func NewAuthHandler(svcCtx *svc.ServiceContext, secureCookies bool) *AuthHandler {
	return &AuthHandler{svcCtx: svcCtx, secureCookies: secureCookies}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/register", request.WithValidatedJSON[logic.Credentials](16<<10, h.register))
	r.Post("/login", request.WithValidatedJSON[logic.Credentials](16<<10, h.login))
	r.Post("/logout", h.logout)
	r.With(authmid.RequireUser).Get("/me", h.me)
}

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request, req *logic.Credentials) {
	l := logic.NewAuthLogic(r.Context(), h.svcCtx)
	_, code, msg := l.Register(req)
	if code != response.CodeSuccess {
		response.Reply(r.Context(), w, nil, code, msg)
		return
	}

	l2 := logic.NewAuthLogic(r.Context(), h.svcCtx)
	data, code, msg := l2.Login(req)
	if code != response.CodeSuccess {
		response.Reply(r.Context(), w, nil, code, msg)
		return
	}
	result := data.(*logic.LoginResult)
	h.setSessionCookie(w, result.Token)
	response.Reply(r.Context(), w, result.User, response.CodeCreated, "")
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request, req *logic.Credentials) {
	l := logic.NewAuthLogic(r.Context(), h.svcCtx)
	data, code, msg := l.Login(req)
	if code != response.CodeSuccess {
		response.Reply(r.Context(), w, nil, code, msg)
		return
	}
	result := data.(*logic.LoginResult)
	h.setSessionCookie(w, result.Token)
	response.Reply(r.Context(), w, result.User, response.CodeSuccess, "")
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	var code response.Code
	var msg string
	if cookie, err := r.Cookie(auth.SessionCookie); err == nil {
		l := logic.NewAuthLogic(r.Context(), h.svcCtx)
		_, code, msg = l.Logout(cookie.Value)
	}
	h.clearSessionCookie(w)
	if code == 0 {
		code = response.CodeNoContent
	}
	response.Reply(r.Context(), w, nil, code, msg)
}

func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	user := authmid.GetUser(r.Context())
	response.Reply(r.Context(), w, user, response.CodeSuccess, "")
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string) {
	const sessionDuration = 30 * 24 * time.Hour
	http.SetCookie(w, &http.Cookie{Name: auth.SessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode, MaxAge: int(sessionDuration.Seconds()), Expires: time.Now().Add(sessionDuration)})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: auth.SessionCookie, Value: "", Path: "/", HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
}
