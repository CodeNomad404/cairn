package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"golang.org/x/crypto/bcrypt"

	"cairn.cn/internal/auth"
	"cairn.cn/internal/auth/logic"
)

type fakeUserRepo struct {
	user *auth.User
}

func (f *fakeUserRepo) Create(_ context.Context, u *auth.User) error {
	u.ID = 1
	return nil
}
func (f *fakeUserRepo) GetByUsername(_ context.Context, _ string) (*auth.User, error) {
	return f.user, nil
}
func (f *fakeUserRepo) GetByID(_ context.Context, _ int64) (*auth.User, error) {
	return f.user, nil
}

type fakeSessionRepo struct{}

func (f *fakeSessionRepo) Create(_ context.Context, _ *auth.Session) error { return nil }
func (f *fakeSessionRepo) GetByToken(_ context.Context, _ string) (*auth.Session, error) {
	return nil, nil
}
func (f *fakeSessionRepo) Delete(_ context.Context, _ string) error { return nil }
func (f *fakeSessionRepo) DeleteExpired(_ context.Context) error    { return nil }

func TestLoginSetsCookieAndHidesToken(t *testing.T) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	authLogic := logic.NewAuthLogic(
		&fakeUserRepo{user: &auth.User{ID: 1, Username: "alice", Password: string(hash), Role: auth.RoleUser}},
		&fakeSessionRepo{},
	)
	h := NewAuthHandler(authLogic, false)

	body := strings.NewReader(`{"username":"alice","password":"correct"}`)
	r := httptest.NewRequest(http.MethodPost, "/login", body)
	w := httptest.NewRecorder()

	h.login(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}

	// 必须设置 Cookie。
	cookies := w.Result().Cookies()
	var sessionCookie *http.Cookie
	for _, c := range cookies {
		if c.Name == auth.SessionCookie {
			sessionCookie = c
			break
		}
	}
	if sessionCookie == nil {
		t.Fatal("Set-Cookie not found for cairn_session")
	}
	if sessionCookie.Value == "" {
		t.Fatal("session cookie has empty value")
	}
	if !sessionCookie.HttpOnly {
		t.Fatal("session cookie not HttpOnly")
	}

	// Token 绝不能出现在 JSON 响应体中。
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	for k := range resp {
		if k == "token" {
			t.Fatal("response body must not expose token")
		}
	}
	if data, ok := resp["data"].(map[string]any); ok {
		if _, exists := data["token"]; exists {
			t.Fatal("response data must not expose token")
		}
	}
}
