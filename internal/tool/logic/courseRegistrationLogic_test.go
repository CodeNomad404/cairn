package logic

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
)

type fakeCourseRegistrationCredentialsRepo struct {
	credentials *toolmodel.CourseRegistrationCredentials
	upsertCount int
}

func (f *fakeCourseRegistrationCredentialsRepo) GetByUserID(
	_ context.Context,
	_ int64,
) (*toolmodel.CourseRegistrationCredentials, error) {
	if f.credentials == nil {
		return nil, tool.ErrCourseRegistrationCredentialsNotFound
	}
	credentials := *f.credentials
	return &credentials, nil
}

func (f *fakeCourseRegistrationCredentialsRepo) Upsert(
	_ context.Context,
	credentials *toolmodel.CourseRegistrationCredentials,
) error {
	copy := *credentials
	f.credentials = &copy
	f.upsertCount++
	return nil
}

func TestCourseRegistrationLogin(t *testing.T) {
	var casSubmitted atomic.Bool
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/home/":
			if cookie, err := r.Cookie("mis_session"); err == nil && cookie.Value == "logged-in" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("MIS home"))
				return
			}
			http.Redirect(w, r, server.URL+"/auth/login/?next=/home/", http.StatusFound)
		case "/auth/login/":
			if r.Method == http.MethodGet {
				_, _ = w.Write([]byte(`<html><body>
<form id="login" method="post">
<input type="hidden" name="next" value="/home/">
<input type="hidden" name="csrfmiddlewaretoken" value="csrf-token">
<input type="text" name="loginname">
<input type="password" name="password">
<img class="captcha" src="/image/captcha-token/">
<input type="hidden" name="captcha_0" value="captcha-token">
<input type="text" name="captcha_1">
</form></body></html>`))
				return
			}
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("csrfmiddlewaretoken") != "csrf-token" ||
				r.Form.Get("captcha_0") != "captcha-token" ||
				r.Form.Get("captcha_1") != "50" ||
				r.Form.Get("loginname") != "24280000" ||
				r.Form.Get("password") != "school-password" {
				t.Fatalf("unexpected CAS form fields: %v", r.Form)
			}
			casSubmitted.Store(true)
			http.SetCookie(w, &http.Cookie{Name: "mis_session", Value: "logged-in", Path: "/"})
			http.Redirect(w, r, server.URL+"/home/", http.StatusFound)
		case "/image/captcha-token/":
			w.Header().Set("Content-Type", "image/png")
			_, _ = w.Write([]byte("captcha-image"))
		case "/predict":
			var request struct {
				Username string `json:"username"`
				Password string `json:"password"`
				TypeID   int    `json:"typeid"`
				Image    string `json:"image"`
			}
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				t.Fatal(err)
			}
			if request.Username != "tujian-user" || request.Password != "tujian-password" || request.TypeID != 11 {
				t.Fatalf("unexpected Tujian request: %#v", request)
			}
			image, err := base64.StdEncoding.DecodeString(request.Image)
			if err != nil || string(image) != "captcha-image" {
				t.Fatalf("unexpected captcha image %q, err %v", image, err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"data":{"result":"50"}}`))
		case "/module/module/10/":
			if _, err := r.Cookie("mis_session"); err != nil {
				http.Error(w, "missing MIS session", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`<a href="/academic-entry/">教务系统</a>`))
		case "/academic-entry/":
			_, _ = w.Write([]byte("AA home"))
		case "/course_selection/courseselecttask/selects/":
			_, _ = w.Write([]byte(`<div>选课已关闭：<span>无当前阶段</span></div>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	repo := &fakeCourseRegistrationCredentialsRepo{}
	l := testCourseRegistrationLogic(t, server.URL, repo)
	result, code, err := l.Login(
		context.Background(), 7, "24280000", "school-password", "tujian-user", "tujian-password",
	)
	if err != nil {
		t.Fatal(err)
	}
	if code != httpx.CodeSuccess {
		t.Fatalf("code = %d, want %d", code, httpx.CodeSuccess)
	}
	if !result.LoggedIn {
		t.Fatalf("result = %#v", result)
	}
	if !casSubmitted.Load() {
		t.Fatal("CAS login form was not submitted")
	}
	l.mu.RLock()
	_, stored := l.sessions[7]
	l.mu.RUnlock()
	if !stored {
		t.Fatal("authenticated session was not stored")
	}
	if repo.upsertCount != 1 || repo.credentials == nil || repo.credentials.StudentID != "24280000" {
		t.Fatalf("credentials were not saved after login: %#v", repo.credentials)
	}

	opened, code, err := l.OpenCourseSelection(context.Background(), 7)
	if err != nil {
		t.Fatal(err)
	}
	if code != httpx.CodeSuccess || !opened.Entered || opened.SelectionOpen {
		t.Fatalf("open result = %#v, code = %d", opened, code)
	}

	result, code, err = l.Login(context.Background(), 7, "", "", "", "")
	if err != nil || !result.LoggedIn {
		t.Fatalf("login with saved credentials: result %#v, err %v", result, err)
	}
	if code != httpx.CodeSuccess {
		t.Fatalf("code = %d, want %d", code, httpx.CodeSuccess)
	}
	if repo.upsertCount != 1 {
		t.Fatalf("saved credentials were written again: upsert count = %d", repo.upsertCount)
	}
}

func TestCourseRegistrationLogin_TujianFailure(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/home/":
			http.Redirect(w, r, server.URL+"/auth/login/", http.StatusFound)
		case "/auth/login/":
			_, _ = w.Write([]byte(`<form id="login">
<input name="next" value="/home/"><input name="csrfmiddlewaretoken" value="csrf">
<img class="captcha" src="/image/token/"><input name="captcha_0" value="token">
</form>`))
		case "/image/token/":
			_, _ = w.Write([]byte("captcha"))
		case "/predict":
			_, _ = w.Write([]byte(`{"success":false,"message":"bad credentials"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	repo := &fakeCourseRegistrationCredentialsRepo{}
	l := testCourseRegistrationLogic(t, server.URL, repo)
	_, code, err := l.Login(context.Background(), 7, "student", "password", "user", "bad-password")
	if err == nil {
		t.Fatal("Login error = nil, want captcha recognition error")
	}
	if code != httpx.CodeValidationFailed {
		t.Fatalf("code = %d, want %d", code, httpx.CodeValidationFailed)
	}
	if repo.upsertCount != 0 {
		t.Fatalf("failed credentials were saved: upsert count = %d", repo.upsertCount)
	}
}

func TestCourseRegistrationLogin_NoSavedCredentials(t *testing.T) {
	l := NewCourseRegistrationLogic(
		&fakeCourseRegistrationCredentialsRepo{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	_, code, err := l.Login(context.Background(), 7, "", "", "", "")
	if err == nil {
		t.Fatal("Login error = nil")
	}
	if code != httpx.CodeNotFound {
		t.Fatalf("code = %d, want %d", code, httpx.CodeNotFound)
	}
}

func TestOpenCourseSelection_RequiresBJTULogin(t *testing.T) {
	l := NewCourseRegistrationLogic(
		&fakeCourseRegistrationCredentialsRepo{},
		slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	_, code, err := l.OpenCourseSelection(context.Background(), 7)
	if err == nil {
		t.Fatal("OpenCourseSelection error = nil")
	}
	if code != httpx.CodeUnauthorized {
		t.Fatalf("code = %d, want %d", code, httpx.CodeUnauthorized)
	}
}

func TestParseCASLoginForm_RequiresStableFields(t *testing.T) {
	_, err := parseCASLoginForm(strings.NewReader(`<form id="login"><input name="next" value="/home/"></form>`))
	if err == nil {
		t.Fatal("parseCASLoginForm error = nil")
	}
}

func testCourseRegistrationLogic(
	t *testing.T,
	serverURL string,
	repo tool.CourseRegistrationCredentialsRepo,
) *CourseRegistrationLogic {
	t.Helper()
	parsed, err := url.Parse(serverURL)
	if err != nil {
		t.Fatal(err)
	}
	l := NewCourseRegistrationLogic(repo, slog.New(slog.NewTextHandler(io.Discard, nil)))
	l.misHomeURL = serverURL + "/home/"
	l.misCourseModuleURL = serverURL + "/module/module/10/"
	l.aaCourseSelectionURL = serverURL + "/course_selection/courseselecttask/selects/"
	l.casHost = parsed.Host
	l.aaHost = parsed.Host
	l.tujianURL = serverURL + "/predict"
	l.requestTimeout = 2 * time.Second
	return l
}
