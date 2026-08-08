package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"cairn.cn/internal/auth"
	"cairn.cn/internal/auth/logic"
	authmid "cairn.cn/internal/auth/middleware"
	authmodel "cairn.cn/internal/auth/model"
)

type fakeUserRepo struct {
	user     *authmodel.User
	missOnce bool // 首次 GetByUsername 无匹配（注册的存在性检查），之后返回用户
}

func (f *fakeUserRepo) Create(_ context.Context, u *authmodel.User) error {
	u.ID = 1
	return nil
}
func (f *fakeUserRepo) GetByUsername(_ context.Context, _ string) (*authmodel.User, error) {
	if f.missOnce {
		f.missOnce = false
		return nil, nil
	}
	return f.user, nil
}
func (f *fakeUserRepo) GetByID(_ context.Context, _ int64) (*authmodel.User, error) {
	return f.user, nil
}

type fakeSessionRepo struct{}

func (f *fakeSessionRepo) Create(_ context.Context, _ *authmodel.Session) error { return nil }
func (f *fakeSessionRepo) GetByToken(_ context.Context, _ string) (*authmodel.Session, error) {
	return nil, nil
}
func (f *fakeSessionRepo) Delete(_ context.Context, _ string) error { return nil }
func (f *fakeSessionRepo) DeleteExpired(_ context.Context) error    { return nil }

func TestLoginSetsCookieAndHidesToken(t *testing.T) {
	createdAt := time.Date(2026, time.July, 19, 8, 0, 0, 0, time.UTC)
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	authLogic := logic.NewAuthLogic(
		&fakeUserRepo{user: &authmodel.User{ID: 1, Username: "alice", Password: string(hash), Role: auth.RoleUser, CreatedAt: createdAt}},
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
	assertUserResultData(t, resp["data"], logic.UserResult{ID: 1, Username: "alice", Role: auth.RoleUser, CreatedAt: createdAt})
}

// assertUserResultData 断言 API 输出的 data 恰好是 UserResult 的 4 字段精确契约：
// id/username/role/created_at，不允许任何其他 key（含 password、token、session），
// 并校验各字段值与期望一致。
func assertUserResultData(t *testing.T, data any, want logic.UserResult) {
	t.Helper()
	obj, ok := data.(map[string]any)
	if !ok {
		t.Fatalf("data = %#v, want object", data)
	}
	for k := range obj {
		switch k {
		case "id", "username", "role", "created_at":
		default:
			t.Fatalf("response data exposes unexpected key %q: %v", k, obj)
		}
	}
	for _, k := range []string{"id", "username", "role", "created_at"} {
		if _, exists := obj[k]; !exists {
			t.Fatalf("response data missing %q: %v", k, obj)
		}
	}
	if len(obj) != 4 {
		t.Fatalf("response data has %d keys, want exactly 4: %v", len(obj), obj)
	}

	if got := int64(obj["id"].(float64)); got != want.ID {
		t.Fatalf("id = %d, want %d", got, want.ID)
	}
	if got := obj["username"].(string); got != want.Username {
		t.Fatalf("username = %q, want %q", got, want.Username)
	}
	if got := obj["role"].(string); got != want.Role {
		t.Fatalf("role = %q, want %q", got, want.Role)
	}
	if got := obj["created_at"].(string); got != want.CreatedAt.UTC().Format(time.RFC3339) {
		t.Fatalf("created_at = %q, want %q", got, want.CreatedAt.UTC().Format(time.RFC3339))
	}
}

func TestRegisterSetsCookieAndReturnsUserResult(t *testing.T) {
	createdAt := time.Date(2026, time.July, 19, 8, 0, 0, 0, time.UTC)
	hash, _ := bcrypt.GenerateFromPassword([]byte("correct"), bcrypt.DefaultCost)
	authLogic := logic.NewAuthLogic(
		&fakeUserRepo{
			missOnce: true,
			user:     &authmodel.User{ID: 1, Username: "alice", Password: string(hash), Role: auth.RoleUser, CreatedAt: createdAt},
		},
		&fakeSessionRepo{},
	)
	h := NewAuthHandler(authLogic, false)

	body := strings.NewReader(`{"username":"alice","password":"correct"}`)
	r := httptest.NewRequest(http.MethodPost, "/register", body)
	w := httptest.NewRecorder()

	h.register(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	// 注册成功后自动登录，必须设置会话 Cookie。
	if len(w.Result().Cookies()) == 0 {
		t.Fatal("Set-Cookie not found after register")
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	assertUserResultData(t, resp["data"], logic.UserResult{ID: 1, Username: "alice", Role: auth.RoleUser, CreatedAt: createdAt})
}

func TestMeReturnsUserResult(t *testing.T) {
	createdAt := time.Date(2026, time.July, 19, 8, 0, 0, 0, time.UTC)
	authLogic := logic.NewAuthLogic(
		&fakeUserRepo{user: &authmodel.User{ID: 1, Username: "alice", Role: auth.RoleUser, CreatedAt: createdAt}},
		&fakeSessionRepo{},
	)
	h := NewAuthHandler(authLogic, false)

	r := httptest.NewRequest(http.MethodGet, "/me", nil)
	r = r.WithContext(authmid.WithPrincipal(r.Context(), &auth.Principal{ID: 1, Username: "alice", Role: auth.RoleUser}))
	w := httptest.NewRecorder()

	h.me(w, r)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	var resp map[string]any
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	assertUserResultData(t, resp["data"], logic.UserResult{ID: 1, Username: "alice", Role: auth.RoleUser, CreatedAt: createdAt})
}
