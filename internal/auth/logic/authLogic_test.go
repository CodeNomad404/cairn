package logic

import (
	"context"
	"errors"
	"testing"
	"time"

	"cairn.cn/internal/auth"
	authmodel "cairn.cn/internal/auth/model"
	"cairn.cn/internal/httpx"
)

type fakeUserRepo struct {
	user *authmodel.User
	err  error
}

func (f *fakeUserRepo) Create(_ context.Context, _ *authmodel.User) error { return nil }
func (f *fakeUserRepo) GetByUsername(_ context.Context, _ string) (*authmodel.User, error) {
	return f.user, f.err
}
func (f *fakeUserRepo) GetByID(_ context.Context, _ int64) (*authmodel.User, error) {
	return f.user, f.err
}

type fakeSessionRepo struct {
	session *authmodel.Session
	deleted string
}

func (f *fakeSessionRepo) Create(_ context.Context, _ *authmodel.Session) error { return nil }
func (f *fakeSessionRepo) GetByToken(_ context.Context, _ string) (*authmodel.Session, error) {
	return f.session, nil
}
func (f *fakeSessionRepo) Delete(_ context.Context, token string) error {
	f.deleted = token
	return nil
}
func (f *fakeSessionRepo) DeleteExpired(_ context.Context) error { return nil }

func TestGetPrincipalByToken(t *testing.T) {
	l := NewAuthLogic(
		&fakeUserRepo{user: &authmodel.User{ID: 7, Username: "alice", Role: auth.RoleUser}},
		&fakeSessionRepo{session: &authmodel.Session{
			Token:     "t",
			UserID:    7,
			ExpiresAt: time.Now().Add(time.Hour),
		}},
	)

	principal, err := l.GetPrincipalByToken(context.Background(), "t")
	if err != nil {
		t.Fatal(err)
	}
	if principal.ID != 7 || principal.Username != "alice" || principal.Role != auth.RoleUser {
		t.Fatalf("principal = %#v", principal)
	}
}

func TestGetPrincipalByToken_ExpiredSessionDeleted(t *testing.T) {
	sessions := &fakeSessionRepo{session: &authmodel.Session{
		Token:     "t",
		UserID:    7,
		ExpiresAt: time.Now().Add(-time.Hour),
	}}
	l := NewAuthLogic(&fakeUserRepo{user: &authmodel.User{ID: 7}}, sessions)

	if _, err := l.GetPrincipalByToken(context.Background(), "t"); err == nil {
		t.Fatal("want error for expired session")
	}
	if sessions.deleted != "t" {
		t.Fatalf("expired session not deleted, deleted = %q", sessions.deleted)
	}
}

func TestMeReturnsUserResult(t *testing.T) {
	createdAt := time.Date(2026, time.July, 19, 8, 0, 0, 0, time.UTC)
	l := NewAuthLogic(
		&fakeUserRepo{user: &authmodel.User{ID: 7, Username: "alice", Role: auth.RoleUser, CreatedAt: createdAt}},
		&fakeSessionRepo{},
	)

	result, code, err := l.Me(context.Background(), &auth.Principal{ID: 7})
	if err != nil {
		t.Fatal(err)
	}
	if code != httpx.CodeSuccess {
		t.Fatalf("code = %d, want 0", code)
	}
	if result.ID != 7 || result.Username != "alice" || result.Role != auth.RoleUser {
		t.Fatalf("result = %#v", result)
	}
	if !result.CreatedAt.Equal(createdAt) {
		t.Fatalf("created_at = %v, want %v", result.CreatedAt, createdAt)
	}
}

func TestMeNilPrincipal(t *testing.T) {
	l := NewAuthLogic(&fakeUserRepo{}, &fakeSessionRepo{})

	_, code, err := l.Me(context.Background(), nil)
	if err == nil {
		t.Fatal("want error for nil principal")
	}
	if code != httpx.CodeUnauthorized {
		t.Fatalf("code = %d, want 401", code)
	}
}

func TestMeQueryErrorHidesInternalDetail(t *testing.T) {
	l := NewAuthLogic(
		&fakeUserRepo{err: errors.New("database password leaked")},
		&fakeSessionRepo{},
	)

	_, code, err := l.Me(context.Background(), &auth.Principal{ID: 7})
	if err == nil {
		t.Fatal("want error")
	}
	if code != httpx.CodeInternalError {
		t.Fatalf("code = %d, want 500", code)
	}
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("err = %T, want *AppError", err)
	}
	if appErr.Message != "failed to load user" {
		t.Fatalf("message = %q, want fixed safe message", appErr.Message)
	}
}
