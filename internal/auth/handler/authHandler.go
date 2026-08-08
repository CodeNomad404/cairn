package handler

import (
	"errors"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/auth"
	"cairn.cn/internal/auth/logic"
	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/svc"
	"cairn.cn/internal/util"
)

// add
// AuthHandler exposes auth endpoints over HTTP.
type AuthHandler struct {
	svcCtx        *svc.ServiceContext
	secureCookies bool
}

func NewAuthHandler(svcCtx *svc.ServiceContext, secureCookies bool) *AuthHandler {
	return &AuthHandler{svcCtx: svcCtx, secureCookies: secureCookies}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/register", util.ParseBody[RegisterRequest](16<<10, h.register))
	r.Post("/login", util.ParseBody[LoginRequest](16<<10, h.login))
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

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request, req *RegisterRequest) {
	l := logic.NewAuthLogic(h.svcCtx)
	_, code, msg := l.Register(r.Context(), req.Username, req.Password)
	if code != util.CodeSuccess {
		util.Reply(r.Context(), w, nil, code, msg)
		return
	}

	data, code, msg := l.Login(r.Context(), req.Username, req.Password)
	if code != util.CodeSuccess {
		util.Reply(r.Context(), w, nil, code, msg)
		return
	}
	result := data.(*logic.LoginResult)
	h.setSessionCookie(w, result.Token)
	util.Reply(r.Context(), w, result.User, util.CodeCreated, "")
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

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request, req *LoginRequest) {
	l := logic.NewAuthLogic(h.svcCtx)
	data, code, msg := l.Login(r.Context(), req.Username, req.Password)
	if code != util.CodeSuccess {
		util.Reply(r.Context(), w, nil, code, msg)
		return
	}
	result := data.(*logic.LoginResult)
	h.setSessionCookie(w, result.Token)
	util.Reply(r.Context(), w, result.User, util.CodeSuccess, "")
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	var code util.Code
	var msg string
	if cookie, err := r.Cookie(auth.SessionCookie); err == nil {
		l := logic.NewAuthLogic(h.svcCtx)
		_, code, msg = l.Logout(r.Context(), cookie.Value)
	}
	h.clearSessionCookie(w)
	if code == 0 {
		code = util.CodeNoContent
	}
	util.Reply(r.Context(), w, nil, code, msg)
}

func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	user := authmid.GetUser(r.Context())
	util.Reply(r.Context(), w, user, util.CodeSuccess, "")
}
