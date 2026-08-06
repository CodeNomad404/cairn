package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cairn.cn/internal/auth"
	"cairn.cn/internal/web"
)

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name       string
		user       *auth.User
		wantStatus int
		wantCode   string
	}{
		{name: "unauthenticated", wantStatus: http.StatusUnauthorized, wantCode: "unauthorized"},
		{name: "wrong role", user: &auth.User{Role: auth.RoleUser}, wantStatus: http.StatusForbidden, wantCode: "forbidden"},
		{name: "admin", user: &auth.User{Role: auth.RoleAdmin}, wantStatus: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodPost, "/posts", nil)
			if tt.user != nil {
				request = request.WithContext(context.WithValue(request.Context(), userKey, tt.user))
			}
			recorder := httptest.NewRecorder()
			RequireRole(auth.RoleAdmin)(next).ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if tt.wantCode == "" {
				return
			}
			var response struct {
				Error web.APIError `json:"error"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
				t.Fatal(err)
			}
			if response.Error.Code != tt.wantCode {
				t.Fatalf("error code = %q, want %q", response.Error.Code, tt.wantCode)
			}
		})
	}
}
