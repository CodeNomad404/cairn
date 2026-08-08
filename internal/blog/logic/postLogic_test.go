package logic

import (
	"context"
	"errors"
	"strings"
	"testing"

	"cairn.cn/internal/blog"
	blogmodel "cairn.cn/internal/blog/model"
	"cairn.cn/internal/httpx"
)

type fakePostRepo struct {
	deleteSlug string
	deleteErr  error
}

func (f *fakePostRepo) Create(context.Context, *blogmodel.Post) error { return nil }
func (f *fakePostRepo) GetBySlug(context.Context, string) (*blogmodel.Post, error) {
	return nil, nil
}
func (f *fakePostRepo) ListPublished(context.Context, int, int) ([]blogmodel.Post, error) {
	return nil, nil
}
func (f *fakePostRepo) ListPublishedMetadata(context.Context) ([]blogmodel.Post, error) {
	return nil, nil
}
func (f *fakePostRepo) Delete(_ context.Context, slug string) error {
	f.deleteSlug = slug
	return f.deleteErr
}

func TestPostLogicDelete(t *testing.T) {
	repo := &fakePostRepo{}
	l := &PostLogic{postRepo: repo}

	code, err := l.Delete(context.Background(), "hello")
	if err != nil {
		t.Fatal(err)
	}
	if code != httpx.CodeNoContent {
		t.Fatalf("code = %d, want %d", code, httpx.CodeNoContent)
	}
	if repo.deleteSlug != "hello" {
		t.Fatalf("repo got slug %q, want %q", repo.deleteSlug, "hello")
	}
}

func TestPostLogicDeleteNotFound(t *testing.T) {
	repo := &fakePostRepo{deleteErr: blog.ErrPostNotFound}
	l := &PostLogic{postRepo: repo}

	code, err := l.Delete(context.Background(), "missing")
	if code != httpx.CodeNotFound {
		t.Fatalf("code = %d, want %d", code, httpx.CodeNotFound)
	}
	if !errors.Is(err, blog.ErrPostNotFound) {
		t.Fatalf("err = %v, want errors.Is(err, blog.ErrPostNotFound)", err)
	}
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("err = %v, want *httpx.AppError", err)
	}
	if appErr.Message != "post not found" {
		t.Fatalf("message = %q, want %q", appErr.Message, "post not found")
	}
}

func TestPostLogicDeleteDatabaseError(t *testing.T) {
	underlying := errors.New("database unavailable: secret detail")
	repo := &fakePostRepo{deleteErr: underlying}
	l := &PostLogic{postRepo: repo}

	code, err := l.Delete(context.Background(), "hello")
	if code != httpx.CodeInternalError {
		t.Fatalf("code = %d, want %d", code, httpx.CodeInternalError)
	}
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("err = %v, want *httpx.AppError", err)
	}
	if appErr.Message != "failed to delete post" {
		t.Fatalf("message = %q, want %q", appErr.Message, "failed to delete post")
	}
	if !errors.Is(err, underlying) {
		t.Fatalf("err = %v, want errors.Is(err, underlying) 保留 Cause", err)
	}
	if strings.Contains(appErr.Message, "secret") {
		t.Fatalf("对外消息泄露内部错误: %q", appErr.Message)
	}
}
