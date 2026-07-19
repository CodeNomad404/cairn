package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/auth"
	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/auth/service"
	"cairn.cn/internal/web"
)

type AuthHandler struct {
	svc           *service.AuthService
	secureCookies bool
}

func NewAuthHandler(svc *service.AuthService, secureCookies bool) *AuthHandler {
	return &AuthHandler{svc: svc, secureCookies: secureCookies}
}

func (h *AuthHandler) RegisterRoutes(r chi.Router) {
	r.Post("/register", h.register)
	r.Post("/login", h.login)
	r.Post("/logout", h.logout)
	r.Get("/me", h.me)
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func (h *AuthHandler) register(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeCredentials(w, r)
	if !ok {
		return
	}
	if _, err := h.svc.Register(r.Context(), input.Username, input.Password); err != nil {
		web.WriteError(w, http.StatusUnprocessableEntity, "registration_failed", err.Error())
		return
	}
	token, user, err := h.svc.Login(r.Context(), input.Username, input.Password)
	if err != nil {
		web.WriteError(w, http.StatusInternalServerError, "internal_error", "failed to create session")
		return
	}
	h.setSessionCookie(w, token)
	web.WriteJSON(w, http.StatusCreated, user)
}

func (h *AuthHandler) login(w http.ResponseWriter, r *http.Request) {
	input, ok := decodeCredentials(w, r)
	if !ok {
		return
	}
	token, user, err := h.svc.Login(r.Context(), input.Username, input.Password)
	if err != nil {
		web.WriteError(w, http.StatusUnauthorized, "invalid_credentials", "invalid username or password")
		return
	}
	h.setSessionCookie(w, token)
	web.WriteJSON(w, http.StatusOK, user)
}

func (h *AuthHandler) logout(w http.ResponseWriter, r *http.Request) {
	if cookie, err := r.Cookie(auth.SessionCookie); err == nil {
		_ = h.svc.Logout(r.Context(), cookie.Value)
	}
	h.clearSessionCookie(w)
	w.WriteHeader(http.StatusNoContent)
}

func (h *AuthHandler) me(w http.ResponseWriter, r *http.Request) {
	user := authmid.GetUser(r.Context())
	if user == nil {
		web.WriteError(w, http.StatusUnauthorized, "unauthorized", "login required")
		return
	}
	web.WriteJSON(w, http.StatusOK, user)
}

func decodeCredentials(w http.ResponseWriter, r *http.Request) (credentials, bool) {
	var input credentials
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 16<<10)).Decode(&input); err != nil {
		web.WriteError(w, http.StatusBadRequest, "invalid_request", "username and password are required")
		return credentials{}, false
	}
	return input, true
}

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string) {
	const sessionDuration = 30 * 24 * time.Hour
	http.SetCookie(w, &http.Cookie{Name: auth.SessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode, MaxAge: int(sessionDuration.Seconds()), Expires: time.Now().Add(sessionDuration)})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: auth.SessionCookie, Value: "", Path: "/", HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
}
