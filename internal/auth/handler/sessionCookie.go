package handler

import (
	"net/http"
	"time"

	"cairn.cn/internal/auth"
)

func (h *AuthHandler) setSessionCookie(w http.ResponseWriter, token string) {
	const sessionDuration = 30 * 24 * time.Hour
	http.SetCookie(w, &http.Cookie{Name: auth.SessionCookie, Value: token, Path: "/", HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode, MaxAge: int(sessionDuration.Seconds()), Expires: time.Now().Add(sessionDuration)})
}

func (h *AuthHandler) clearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{Name: auth.SessionCookie, Value: "", Path: "/", HttpOnly: true, Secure: h.secureCookies, SameSite: http.SameSiteLaxMode, MaxAge: -1, Expires: time.Unix(1, 0)})
}
