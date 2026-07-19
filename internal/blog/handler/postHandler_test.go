package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/blog"
	"cairn.cn/internal/web"
)

type fakePostSource struct {
	posts  []blog.Post
	post   *blog.Post
	err    error
	limit  int
	offset int
}

func (f *fakePostSource) ListPublished(_ context.Context, limit, offset int) ([]blog.Post, error) {
	f.limit = limit
	f.offset = offset
	return f.posts, f.err
}

func (f *fakePostSource) GetBySlug(context.Context, string) (*blog.Post, error) {
	return f.post, f.err
}

func (f *fakePostSource) Publish(context.Context, string, string, string) (*blog.Post, error) {
	return f.post, f.err
}

func apiRouter(source *fakePostSource) http.Handler {
	r := chi.NewRouter()
	r.Route("/api/v1", NewPostHandler(source).RegisterRoutes)
	return r
}

func TestPostList(t *testing.T) {
	publishedAt := time.Date(2026, time.July, 19, 8, 0, 0, 0, time.UTC)
	source := &fakePostSource{posts: []blog.Post{
		{ID: 2, Slug: "second", Title: "Second", PublishedAt: &publishedAt},
		{ID: 1, Slug: "first", Title: "First", PublishedAt: &publishedAt},
	}}
	recorder := httptest.NewRecorder()
	apiRouter(source).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/posts?page=2&page_size=1", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if source.limit != 2 || source.offset != 1 {
		t.Fatalf("pagination = limit %d, offset %d", source.limit, source.offset)
	}

	var response postListResponse
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Page != 2 || response.PageSize != 1 || !response.HasMore {
		t.Fatalf("response pagination = %#v", response)
	}
	if len(response.Items) != 1 || response.Items[0].Slug != "second" {
		t.Fatalf("items = %#v", response.Items)
	}
}

func TestPostListRejectsInvalidPage(t *testing.T) {
	recorder := httptest.NewRecorder()
	apiRouter(&fakePostSource{}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/posts?page=0", nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
	var response struct {
		Error web.APIError `json:"error"`
	}
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Error.Code != "invalid_request" {
		t.Fatalf("error = %#v", response.Error)
	}
}

func TestPostDetailNotFound(t *testing.T) {
	recorder := httptest.NewRecorder()
	apiRouter(&fakePostSource{err: blog.ErrPostNotFound}).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/posts/missing", nil),
	)

	if recorder.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", recorder.Code)
	}
}

func TestPostDetail(t *testing.T) {
	publishedAt := time.Date(2026, time.July, 19, 8, 0, 0, 0, time.UTC)
	recorder := httptest.NewRecorder()
	apiRouter(&fakePostSource{post: &blog.Post{
		ID:          1,
		Slug:        "first",
		Title:       "First",
		ContentHTML: "<p>Hello</p>",
		PublishedAt: &publishedAt,
	}}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/posts/first", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	var response postDetail
	if err := json.NewDecoder(recorder.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Slug != "first" || response.ContentHTML != "<p>Hello</p>" {
		t.Fatalf("response = %#v", response)
	}
}

func TestPostDetailDatabaseError(t *testing.T) {
	recorder := httptest.NewRecorder()
	apiRouter(&fakePostSource{err: errors.New("database unavailable")}).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/api/v1/posts/first", nil),
	)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}
