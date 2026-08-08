package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/auth"
	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/blog"
	bloglogic "cairn.cn/internal/blog/logic"
	"cairn.cn/internal/httpx"
)

type fakePostRepo struct {
	posts  []blog.Post
	post   *blog.Post
	err    error
	limit  int
	offset int
}

func (f *fakePostRepo) Create(_ context.Context, p *blog.Post) error {
	if f.err != nil {
		return f.err
	}
	if f.post != nil {
		*p = *f.post
	}
	return nil
}
func (f *fakePostRepo) GetBySlug(_ context.Context, _ string) (*blog.Post, error) {
	return f.post, f.err
}
func (f *fakePostRepo) ListPublished(_ context.Context, limit, offset int) ([]blog.Post, error) {
	f.limit = limit
	f.offset = offset
	return f.posts, f.err
}
func (f *fakePostRepo) ListPublishedMetadata(context.Context) ([]blog.Post, error) {
	return nil, nil
}

func apiRouter(repo *fakePostRepo) http.Handler {
	r := chi.NewRouter()
	r.Route("/api/v1", NewPostHandler(bloglogic.NewPostLogic(repo)).RegisterRoutes)
	return r
}

func TestPostList(t *testing.T) {
	publishedAt := time.Date(2026, time.July, 19, 8, 0, 0, 0, time.UTC)
	repo := &fakePostRepo{posts: []blog.Post{
		{ID: 2, Slug: "second", Title: "Second", PublishedAt: &publishedAt},
		{ID: 1, Slug: "first", Title: "First", PublishedAt: &publishedAt},
	}}
	recorder := httptest.NewRecorder()
	apiRouter(repo).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/posts?page=2&page_size=1", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if repo.limit != 2 || repo.offset != 1 {
		t.Fatalf("pagination = limit %d, offset %d", repo.limit, repo.offset)
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Items []struct {
				Slug string `json:"slug"`
			} `json:"items"`
			Page     int  `json:"page"`
			PageSize int  `json:"page_size"`
			HasMore  bool `json:"has_more"`
		} `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != 0 {
		t.Fatalf("code = %d", resp.Code)
	}
	if resp.Data.Page != 2 || resp.Data.PageSize != 1 || !resp.Data.HasMore {
		t.Fatalf("pagination = %+v", resp.Data)
	}
	if len(resp.Data.Items) != 1 || resp.Data.Items[0].Slug != "second" {
		t.Fatalf("items = %#v", resp.Data.Items)
	}
}

func TestPostListRejectsInvalidPage(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"zero", "/api/v1/posts?page=0"},
		{"negative", "/api/v1/posts?page=-1"},
		{"non-numeric", "/api/v1/posts?page=abc"},
		{"page_size_zero", "/api/v1/posts?page_size=0"},
		{"page_size_non_numeric", "/api/v1/posts?page_size=abc"},
		{"page_size_over_limit", "/api/v1/posts?page_size=101"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			apiRouter(&fakePostRepo{}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, tt.url, nil))
			if recorder.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, want 400 (url=%s)", recorder.Code, tt.url)
			}
		})
	}
}

func TestPostListDefaults(t *testing.T) {
	repo := &fakePostRepo{}
	recorder := httptest.NewRecorder()
	apiRouter(repo).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/posts", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if repo.limit != 21 || repo.offset != 0 {
		t.Fatalf("pagination = limit %d, offset %d (want limit=21 offset=0)", repo.limit, repo.offset)
	}
}

func TestPostGetNotFound(t *testing.T) {
	recorder := httptest.NewRecorder()
	apiRouter(&fakePostRepo{err: blog.ErrPostNotFound}).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/posts/missing", nil),
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestPostGet(t *testing.T) {
	publishedAt := time.Date(2026, time.July, 19, 8, 0, 0, 0, time.UTC)
	recorder := httptest.NewRecorder()
	apiRouter(&fakePostRepo{post: &blog.Post{
		ID:          1,
		Slug:        "first",
		Title:       "First",
		ContentHTML: "<p>Hello</p>",
		PublishedAt: &publishedAt,
	}}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/posts/first", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var resp struct {
		Code int `json:"code"`
		Data struct {
			Slug        string `json:"slug"`
			ContentHTML string `json:"content_html"`
		} `json:"data"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Data.Slug != "first" || resp.Data.ContentHTML != "<p>Hello</p>" {
		t.Fatalf("data = %#v", resp.Data)
	}
}

func TestPostGetDatabaseError(t *testing.T) {
	recorder := httptest.NewRecorder()
	apiRouter(&fakePostRepo{err: errors.New("database unavailable")}).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/posts/first", nil),
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}

func TestPostCreate(t *testing.T) {
	publishedAt := time.Date(2026, time.July, 19, 8, 0, 0, 0, time.UTC)
	repo := &fakePostRepo{post: &blog.Post{
		ID:          1,
		Slug:        "hello",
		Title:       "Hello",
		ContentHTML: "<p>Hello</p>",
		PublishedAt: &publishedAt,
	}}
	body := strings.NewReader(`{"slug":"hello","title":"Hello","content_md":"# Hello"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/posts", body)
	w := httptest.NewRecorder()

	handler := NewPostHandler(bloglogic.NewPostLogic(repo))
	handler.create(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}

	var resp struct {
		Code int `json:"code"`
		Data struct {
			Slug        string `json:"slug"`
			ID          int64  `json:"id"`
			ContentHTML string `json:"content_html"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != int(httpx.CodeCreated) {
		t.Fatalf("code = %d", resp.Code)
	}
	if resp.Data.Slug != "hello" || resp.Data.ID != 1 || resp.Data.ContentHTML != "<p>Hello</p>" {
		t.Fatalf("data = %#v", resp.Data)
	}
}

// Route-level auth tests — go through the chi router so RequireRole is exercised.

func TestPostCreateRoute_Unauthenticated(t *testing.T) {
	body := strings.NewReader(`{"slug":"a","title":"A","content_md":"# A"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/posts", body)
	w := httptest.NewRecorder()
	apiRouter(&fakePostRepo{}).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

func TestPostCreateRoute_ForbiddenForUser(t *testing.T) {
	body := strings.NewReader(`{"slug":"a","title":"A","content_md":"# A"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/posts", body)
	r = r.WithContext(authmid.WithUser(r.Context(), &auth.User{ID: 1, Role: auth.RoleUser}))
	w := httptest.NewRecorder()
	apiRouter(&fakePostRepo{}).ServeHTTP(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestPostCreateRoute_AdminAllowed(t *testing.T) {
	publishedAt := time.Date(2026, time.July, 19, 8, 0, 0, 0, time.UTC)
	repo := &fakePostRepo{post: &blog.Post{
		ID: 1, Slug: "a", Title: "A",
		ContentHTML: "<p>A</p>",
		PublishedAt: &publishedAt,
	}}
	body := strings.NewReader(`{"slug":"a","title":"A","content_md":"# A"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/posts", body)
	r = r.WithContext(authmid.WithUser(r.Context(), &auth.User{ID: 1, Role: auth.RoleAdmin}))
	w := httptest.NewRecorder()
	apiRouter(repo).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
}
