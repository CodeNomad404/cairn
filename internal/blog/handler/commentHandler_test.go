package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/blog"
	"cairn.cn/internal/svc"
)

type fakeCommentPostRepo struct{}

func (f *fakeCommentPostRepo) Create(_ context.Context, _ *blog.Post) error { return nil }
func (f *fakeCommentPostRepo) GetBySlug(_ context.Context, _ string) (*blog.Post, error) {
	return &blog.Post{ID: 1}, nil
}
func (f *fakeCommentPostRepo) ListPublished(_ context.Context, _ int, _ int) ([]blog.Post, error) {
	return nil, nil
}
func (f *fakeCommentPostRepo) ListPublishedMetadata(_ context.Context) ([]blog.Post, error) {
	return nil, nil
}

type fakeCommentRepo struct{}

func (f *fakeCommentRepo) Create(_ context.Context, c *blog.Comment) error {
	c.ID = 1
	return nil
}
func (f *fakeCommentRepo) ListByPostID(_ context.Context, _ int64) ([]blog.Comment, error) {
	return nil, nil
}
func (f *fakeCommentRepo) GetByID(_ context.Context, _ int64) (*blog.Comment, error) {
	return nil, nil
}
func (f *fakeCommentRepo) Delete(_ context.Context, _ int64) error { return nil }

func commentRouter() http.Handler {
	r := chi.NewRouter()
	svcCtx := &svc.ServiceContext{
		PostRepo:    &fakeCommentPostRepo{},
		CommentRepo: &fakeCommentRepo{},
	}
	r.Route("/api/v1", NewCommentHandler(svcCtx).RegisterRoutes)
	return r
}

func TestCommentCreateRoute_Unauthenticated(t *testing.T) {
	// Even with invalid JSON, RequireUser fires first → 401, not 400.
	body := strings.NewReader(`not-json`)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/posts/some-slug/comments", body)
	w := httptest.NewRecorder()
	commentRouter().ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (RequireUser must fire before JSON parsing)", w.Code)
	}
}
