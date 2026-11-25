package middleware

import (
	"net/http"
	"time"

	chimid "github.com/go-chi/chi/v5/middleware"
)

// RequestTimeout retains the ordinary API timeout and extends only binary transfers.
func RequestTimeout(next http.Handler) http.Handler {
	ordinary := chimid.Timeout(30 * time.Second)(next)
	transfer := chimid.Timeout(10 * time.Minute)(next)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if (r.Method == http.MethodPost && r.URL.Path == "/api/v1/tool/transfer/upload") || (r.Method == http.MethodGet && r.URL.Path == "/api/v1/tool/transfer/download") {
			deadline := time.Now().Add(10 * time.Minute)
			controller := http.NewResponseController(w)
			_ = controller.SetReadDeadline(deadline)
			_ = controller.SetWriteDeadline(deadline)
			transfer.ServeHTTP(w, r)
			return
		}
		ordinary.ServeHTTP(w, r)
	})
}
