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
	bloglogic "cairn.cn/internal/blog/logic"
	blogmodel "cairn.cn/internal/blog/model"
	"cairn.cn/internal/httpx"
)

type fakeCommentPostRepo struct {
	callCount int
}

func (f *fakeCommentPostRepo) Create(_ context.Context, _ *blogmodel.Post) error { return nil }
func (f *fakeCommentPostRepo) GetBySlug(_ context.Context, _ string) (*blogmodel.Post, error) {
	f.callCount++
	return &blogmodel.Post{ID: 1}, nil
}
func (f *fakeCommentPostRepo) ListPublished(_ context.Context, _ int, _ int) ([]blogmodel.Post, error) {
	return nil, nil
}
func (f *fakeCommentPostRepo) ListPublishedMetadata(_ context.Context) ([]blogmodel.Post, error) {
	return nil, nil
}
func (f *fakeCommentPostRepo) Delete(context.Context, string) error { return nil }

type fakeCommentRepo struct {
	comment *blogmodel.Comment
}

func (f *fakeCommentRepo) Create(_ context.Context, c *blogmodel.Comment) error {
	c.ID = 1
	return nil
}
func (f *fakeCommentRepo) ListByPostID(_ context.Context, _ int64) ([]blogmodel.Comment, error) {
	return nil, nil
}
func (f *fakeCommentRepo) GetByID(_ context.Context, _ int64) (*blogmodel.Comment, error) {
	return f.comment, nil
}
func (f *fakeCommentRepo) Delete(_ context.Context, _ int64) error { return nil }

func commentRouter(principal *auth.Principal) http.Handler {
	r := chi.NewRouter()
	if principal != nil {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				next.ServeHTTP(w, r.WithContext(authmid.WithPrincipal(r.Context(), principal)))
			})
		})
	}
	commentLogic := bloglogic.NewCommentLogic(&fakeCommentPostRepo{}, &fakeCommentRepo{})
	r.Route("/api/v1", NewCommentHandler(commentLogic).RegisterRoutes)
	return r
}

func TestCommentCreateRoute_Unauthenticated(t *testing.T) {
	// 即使 JSON 无效，RequireUser 也会先触发 → 401，而不是 400。
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
	commentRouter(&auth.Principal{ID: 7, Username: "alice", Role: auth.RoleUser}).ServeHTTP(w, r)

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
	commentRouter(&auth.Principal{ID: 7, Role: auth.RoleUser}).ServeHTTP(w, r)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// 直接调用 handler 的测试——像路由器一样调用 h.create(w, r)，这样无需经过 chi
// 即可验证解析顺序（先 Path 后 Body）。

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
	return r.WithContext(authmid.WithPrincipal(r.Context(), &auth.Principal{ID: 7, Role: auth.RoleUser}))
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
	// 路径解析先失败：即使是有效 Body 也不得到达 Logic。
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
	if resp.Code != int(httpx.CodeBadRequest) || resp.Message != "参数错误" {
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
	if resp.Code != int(httpx.CodeBadRequest) {
		t.Fatalf("code = %d, want 400", resp.Code)
	}
	if repo.callCount != 0 {
		t.Fatalf("logic called %d times, want 0", repo.callCount)
	}
}

func commentDeleteRequest(t *testing.T, id string, principal *auth.Principal) *http.Request {
	t.Helper()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", id)
	r := httptest.NewRequest(http.MethodDelete, "/api/v1/comments/"+id, nil)
	r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
	return r.WithContext(authmid.WithPrincipal(r.Context(), principal))
}

func commentDeleteHandler(comment *blogmodel.Comment) *CommentHandler {
	return NewCommentHandler(bloglogic.NewCommentLogic(&fakeCommentPostRepo{}, &fakeCommentRepo{comment: comment}))
}

func TestCommentDeleteDirect_AuthorDeletesOwnComment(t *testing.T) {
	h := commentDeleteHandler(&blogmodel.Comment{ID: 5, UserID: 7})
	r := commentDeleteRequest(t, "5", &auth.Principal{ID: 7, Role: auth.RoleUser})
	w := httptest.NewRecorder()

	h.delete(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
}

func TestCommentDeleteDirect_AdminDeletesAnyComment(t *testing.T) {
	h := commentDeleteHandler(&blogmodel.Comment{ID: 5, UserID: 7})
	r := commentDeleteRequest(t, "5", &auth.Principal{ID: 1, Role: auth.RoleAdmin})
	w := httptest.NewRecorder()

	h.delete(w, r)

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", w.Code)
	}
}

func TestCommentDeleteDirect_ForbiddenForOtherUser(t *testing.T) {
	h := commentDeleteHandler(&blogmodel.Comment{ID: 5, UserID: 7})
	r := commentDeleteRequest(t, "5", &auth.Principal{ID: 8, Role: auth.RoleUser})
	w := httptest.NewRecorder()

	h.delete(w, r)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestCommentDeleteDirect_Unauthenticated(t *testing.T) {
	h := commentDeleteHandler(&blogmodel.Comment{ID: 5, UserID: 7})
	r := commentDeleteRequest(t, "5", nil)
	w := httptest.NewRecorder()

	h.delete(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}
