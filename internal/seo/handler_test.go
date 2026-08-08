package seo

import (
	"context"
	"encoding/xml"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/blog"
	bloglogic "cairn.cn/internal/blog/logic"
)

type fakePostRepo struct {
	posts []blog.Post
	err   error
}

func (f fakePostRepo) Create(context.Context, *blog.Post) error { return nil }
func (f fakePostRepo) GetBySlug(context.Context, string) (*blog.Post, error) {
	return nil, nil
}
func (f fakePostRepo) ListPublished(context.Context, int, int) ([]blog.Post, error) {
	return nil, nil
}
func (f fakePostRepo) ListPublishedMetadata(context.Context) ([]blog.Post, error) {
	return f.posts, f.err
}

func seoRouter(repo blog.PostRepo) http.Handler {
	r := chi.NewRouter()
	New(bloglogic.NewPostLogic(repo)).RegisterRoutes(r)
	return r
}

func TestRobots(t *testing.T) {
	recorder := httptest.NewRecorder()
	seoRouter(fakePostRepo{}).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/robots.txt", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "text/plain; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	want := "User-agent: *\nDisallow: /upload\nDisallow: /healthz\n\nSitemap: https://cairn.cn/sitemap.xml\n"
	if recorder.Body.String() != want {
		t.Fatalf("robots.txt = %q, want %q", recorder.Body.String(), want)
	}
}

func TestSitemap(t *testing.T) {
	updatedAt := time.Date(2026, time.July, 5, 2, 3, 4, 0, time.FixedZone("CST", 8*60*60))
	repo := fakePostRepo{posts: []blog.Post{{Slug: "go 与 web", UpdatedAt: updatedAt}}}
	recorder := httptest.NewRecorder()
	seoRouter(repo).ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/xml; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	if !strings.HasPrefix(recorder.Body.String(), xml.Header) {
		t.Fatal("sitemap is missing XML header")
	}

	var result urlSet
	if err := xml.Unmarshal(recorder.Body.Bytes(), &result); err != nil {
		t.Fatalf("unmarshal sitemap: %v", err)
	}
	if result.XMLNS != "http://www.sitemaps.org/schemas/sitemap/0.9" {
		t.Fatalf("xmlns = %q", result.XMLNS)
	}

	wantLocation := "https://cairn.cn/posts/go%20%E4%B8%8E%20web"
	found := false
	for _, entry := range result.URLs {
		if strings.Contains(entry.Location, "/login") || strings.Contains(entry.Location, "/upload") {
			t.Fatalf("private URL in sitemap: %s", entry.Location)
		}
		if entry.Location == wantLocation {
			found = true
			if entry.LastMod != "2026-07-04T18:03:04Z" {
				t.Fatalf("lastmod = %q", entry.LastMod)
			}
		}
	}
	if !found {
		t.Fatalf("missing escaped post URL %q", wantLocation)
	}
}

func TestSitemapDatabaseError(t *testing.T) {
	recorder := httptest.NewRecorder()
	seoRouter(fakePostRepo{err: errors.New("database unavailable")}).ServeHTTP(
		recorder,
		httptest.NewRequest(http.MethodGet, "/sitemap.xml", nil),
	)
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
}
