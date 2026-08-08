package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/auth"
	authmid "cairn.cn/internal/auth/middleware"
	"cairn.cn/internal/blog"
	bloglogic "cairn.cn/internal/blog/logic"
	"cairn.cn/internal/util"
)

type fakeCommentPostRepo struct {
	callCount int
}

func (f *fakeCommentPostRepo) Create(_ context.Context, _ *blog.Post) error { return nil }
func (f *fakeCommentPostRepo) GetBySlug(_ context.Context, _ string) (*blog.Post, error) {
	f.callCount++
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

func commentRouter(user *auth.User) http.Handler {
	r := chi.NewRouter()
	if user != nil {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r.WithContext(authmid.WithUser(r.Context(), user)))
			})
		})
	}
	commentLogic := bloglogic.NewCommentLogic(&fakeCommentPostRepo{}, &fakeCommentRepo{})
	r.Route("/api/v1", NewCommentHandler(commentLogic).RegisterRoutes)
	return r
}

func TestCommentCreateRoute_Unauthenticated(t *testing.T) {
	// Even with invalid JSON, RequireUser fires first → 401, not 400.
	body := strings.NewReader(`not-json`)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/posts/some-slug/comments", body)
	w := httptest.NewRecorder()
	commentRouter(nil).ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401 (RequireUser must fire before JSON parsing)", w.Code)
	}
}

func TestCommentCreateRoute_UserAllowed(t *testing.T) {
	body := strings.NewReader(`{"content":"hello"}`)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/posts/some-slug/comments", body)
	w := httptest.NewRecorder()
	commentRouter(&auth.User{ID: 7, Username: "alice", Role: auth.RoleUser}).ServeHTTP(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	var response struct {
		Data struct {
			UserID   int64  `json:"user_id"`
			Username string `json:"username"`
		} `json:"data"`
	}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}
	if response.Data.UserID != 7 || response.Data.Username != "alice" {
		t.Fatalf("data = %#v", response.Data)
	}
}

func TestCommentDeleteRoute_RejectsInvalidID(t *testing.T) {
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/comments/invalid", nil)
	w := httptest.NewRecorder()
	commentRouter(&auth.User{ID: 7, Role: auth.RoleUser}).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// Direct handler tests — call h.create(w, r) like the router does, so parsing
// order (Path before Body) is exercised without going through chi.

func newCommentHandler() (*CommentHandler, *fakeCommentPostRepo) {
	repo := &fakeCommentPostRepo{}
	return NewCommentHandler(bloglogic.NewCommentLogic(repo, &fakeCommentRepo{})), repo
}

func commentCreateRequest(t *testing.T, slug, body string) *http.Request {
	t.Helper()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("slug", slug)
	r := httptest.NewRequest(http.MethodPost, "/api/v1/posts/some-slug/comments", strings.NewReader(body))
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	return r.WithContext(authmid.WithUser(r.Context(), &auth.User{ID: 7, Role: auth.RoleUser}))
}

func TestCommentCreateDirect_ParsesPathThenBody(t *testing.T) {
	h, repo := newCommentHandler()
	r := commentCreateRequest(t, "some-slug", `{"content":"hello"}`)
	w := httptest.NewRecorder()

	h.create(w, r)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201", w.Code)
	}
	if repo.callCount != 1 {
		t.Fatalf("logic called %d times, want 1", repo.callCount)
	}
}

func TestCommentCreateDirect_MissingSlugSkipsLogic(t *testing.T) {
	h, repo := newCommentHandler()
	// Path parsing fails first: even a valid body must not reach Logic.
	r := commentCreateRequest(t, "   ", `{"content":"hello"}`)
	w := httptest.NewRecorder()

	h.create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != int(util.CodeBadRequest) || resp.Message != "参数错误" {
		t.Fatalf("response = code %d message %q", resp.Code, resp.Message)
	}
	if repo.callCount != 0 {
		t.Fatalf("logic called %d times, want 0", repo.callCount)
	}
}

func TestCommentCreateDirect_BadBodySkipsLogic(t *testing.T) {
	h, repo := newCommentHandler()
	r := commentCreateRequest(t, "some-slug", `not-json`)
	w := httptest.NewRecorder()

	h.create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var resp struct {
		Code int `json:"code"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != int(util.CodeBadRequest) {
		t.Fatalf("code = %d, want 400", resp.Code)
	}
	if repo.callCount != 0 {
		t.Fatalf("logic called %d times, want 0", repo.callCount)
	}
}
