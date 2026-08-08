package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"cairn.cn/internal/auth"
	"cairn.cn/internal/httpx"
)

func TestGetPrincipal(t *testing.T) {
	principal := &auth.Principal{ID: 7, Username: "alice", Role: auth.RoleUser}
	ctx := WithPrincipal(context.Background(), principal)

	if got := GetPrincipal(ctx); got != principal {
		t.Fatalf("GetPrincipal = %v, want %v", got, principal)
	}
	if got := GetPrincipal(context.Background()); got != nil {
		t.Fatalf("GetPrincipal = %v, want nil", got)
	}
	// Context 中必须是 *auth.Principal，不能是 *auth.User。
	if _, ok := ctx.Value(principalKey).(*auth.Principal); !ok {
		t.Fatal("context must store *auth.Principal")
	}
}

func TestRequireUser(t *testing.T) {
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})

	t.Run("authenticated", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		r = r.WithContext(WithPrincipal(r.Context(), &auth.Principal{ID: 1}))
		w := httptest.NewRecorder()
		RequireUser(next).ServeHTTP(w, r)
		if w.Code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", w.Code)
		}
	})

	t.Run("unauthenticated", func(t *testing.T) {
		r := httptest.NewRequest(http.MethodGet, "/", nil)
		w := httptest.NewRecorder()
		RequireUser(next).ServeHTTP(w, r)
		if w.Code != http.StatusUnauthorized {
			t.Fatalf("status = %d, want 401", w.Code)
		}
	})
}

func TestRequireRole(t *testing.T) {
	tests := []struct {
		name       string
		principal  *auth.Principal
		wantStatus int
		wantCode   int
	}{
		{name: "unauthenticated", wantStatus: http.StatusUnauthorized, wantCode: int(httpx.CodeUnauthorized)},
		{name: "wrong role", principal: &auth.Principal{Role: auth.RoleUser}, wantStatus: http.StatusForbidden, wantCode: int(httpx.CodeForbidden)},
		{name: "admin", principal: &auth.Principal{Role: auth.RoleAdmin}, wantStatus: http.StatusNoContent},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodPost, "/posts", nil)
			if tt.principal != nil {
				request = request.WithContext(WithPrincipal(request.Context(), tt.principal))
			}
			recorder := httptest.NewRecorder()
			RequireRole(auth.RoleAdmin)(next).ServeHTTP(recorder, request)

			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
			if tt.wantCode == 0 {
				return
			}
			var resp struct {
				Code    int    `json:"code"`
				Message string `json:"message,omitempty"`
			}
			if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
				t.Fatal(err)
			}
			if resp.Code != tt.wantCode {
				t.Fatalf("error code = %d, want %d", resp.Code, tt.wantCode)
			}
		})
	}
}
