package logic

import (
	"context"
	"crypto/rand"
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

	"github.com/emmansun/gmsm/sm2"

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

func (f *fakeCourseRegistrationCredentialsRepo) BindMIS(ctx context.Context, c *toolmodel.CourseRegistrationCredentials) error {
	if f.credentials != nil {
		c.TujianUsername = f.credentials.TujianUsername
		c.TujianPassword = f.credentials.TujianPassword
	}
	c.BindingVersion++
	return f.Upsert(ctx, c)
}

func TestCourseRegistrationLogin(t *testing.T) {
	privateKey, encodedPublicKey := testSSOKey(t)
	escapedPublicKey := strings.ReplaceAll(encodedPublicKey, "/", `\/`)

	var ssoSubmitted atomic.Bool
	var courseSelected atomic.Bool
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/home/":
			if r.UserAgent() != browserUserAgent {
				http.Error(w, "browser user agent required", http.StatusForbidden)
				return
			}
			if cookie, err := r.Cookie("mis_session"); err == nil && cookie.Value == "logged-in" {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte("MIS home"))
				return
			}
			http.Redirect(w, r, server.URL+"/o/authorize/", http.StatusFound)
		case "/o/authorize/":
			_, _ = w.Write([]byte(`<html><body>
<form id="loginForm" method="post" action="/sso/login">
<input type="text" name="username">
<input type="password" name="password">
<input type="hidden" name="loginType" value="username_password">
<input type="hidden" name="flowId" value="flow-id">
<input type="hidden" name="client_id" value="mis-client">
<input type="hidden" name="redirect_uri" value="/auth/ncallback/">
<input type="hidden" name="response_type" value="code">
<input type="hidden" name="state" value="state-value">
</form>
<script>var ssoConfig = {"sm2":{"enabled":true,"publicKey":"` + escapedPublicKey + `"}};</script>
</body></html>`))
		case "/sso/login":
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Header.Get("Origin") != server.URL || r.Header.Get("Referer") != server.URL+"/o/authorize/" {
				t.Fatalf("unexpected SSO request headers")
			}
			ciphertext, err := base64.StdEncoding.DecodeString(r.Form.Get("password"))
			if err != nil {
				t.Fatalf("decode encrypted password: %v", err)
			}
			plaintext, err := sm2.Decrypt(privateKey, append([]byte{4}, ciphertext...))
			if err != nil {
				t.Fatalf("decrypt password: %v", err)
			}
			if r.Form.Get("username") != "24280000" ||
				r.Form.Get("loginType") != "username_password" ||
				r.Form.Get("flowId") != "flow-id" ||
				string(plaintext) != "school-password" {
				t.Fatalf("unexpected SSO form fields")
			}
			ssoSubmitted.Store(true)
			http.SetCookie(w, &http.Cookie{Name: "mis_session", Value: "logged-in", Path: "/"})
			http.Redirect(w, r, server.URL+"/auth/ncallback/?code=oauth-code", http.StatusFound)
		case "/auth/ncallback/":
			http.Redirect(w, r, server.URL+"/home/", http.StatusFound)
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
			if request.Username != "tujian-user" || request.Password != "tujian-password" ||
				request.TypeID != 16 {
				t.Fatalf("unexpected Tujian request: %#v", request)
			}
			image, err := base64.StdEncoding.DecodeString(request.Image)
			if err != nil || string(image) != "selection-captcha" {
				t.Fatalf("unexpected captcha image %q, err %v", image, err)
			}
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"success":true,"data":{"result":"captcha-answer"}}`))
		case "/module/module/10/":
			if _, err := r.Cookie("mis_session"); err != nil {
				http.Error(w, "missing MIS session", http.StatusUnauthorized)
				return
			}
			_, _ = w.Write([]byte(`<a href="/academic-entry/">教务系统</a>`))
		case "/academic-entry/":
			_, _ = w.Write([]byte("AA home"))
		case "/course_selection/courseselecttask/selects/":
			message := ""
			if courseSelected.Load() {
				message = `<script>message += "选课成功<br/>";</script>`
			}
			_, _ = w.Write([]byte(`<div id="select-submit-btn"></div>
<table><tr><td>ignored</td></tr></table>
<table><tr><th>选择</th></tr><tr>
<td><input value="course-123"></td><td>高等数学</td><td>3</td>
<td>A101020B</td><td></td><td></td><td>张老师</td></tr></table>` + message))
		case "/captcha/refresh/":
			_, _ = w.Write([]byte(`{"key":"selection-key"}`))
		case "/captcha/image/selection-key":
			_, _ = w.Write([]byte("selection-captcha"))
		case "/course_selection/courseselecttask/selects_action/":
			if r.URL.Query().Get("action") != "submit" {
				http.NotFound(w, r)
				return
			}
			if err := r.ParseForm(); err != nil {
				t.Fatal(err)
			}
			if r.Form.Get("checkboxs") != "course-123" ||
				r.Form.Get("hashkey") != "selection-key" ||
				r.Form.Get("answer") != "captcha-answer" {
				t.Fatalf("unexpected selection form: %v", r.Form)
			}
			courseSelected.Store(true)
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
	if !ssoSubmitted.Load() {
		t.Fatal("SSO login form was not submitted")
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
	if code != httpx.CodeSuccess || !opened.Entered || !opened.SelectionOpen {
		t.Fatalf("open result = %#v, code = %d", opened, code)
	}

	listed, code, err := l.ListCourses(context.Background(), 7)
	if err != nil || code != httpx.CodeSuccess || !listed.SelectionOpen || len(listed.Courses) != 1 {
		t.Fatalf("list result = %#v, code = %d, err = %v", listed, code, err)
	}
	if listed.Courses[0].ID != "course-123" || listed.Courses[0].Remaining != 3 {
		t.Fatalf("unexpected course: %#v", listed.Courses[0])
	}

	grabbed, code, err := l.GrabCourses(
		context.Background(), 7, []string{"高等数学", "不存在的课程"},
	)
	if err != nil || code != httpx.CodeSuccess || len(grabbed.Results) != 2 {
		t.Fatalf("grab result = %#v, code = %d, err = %v", grabbed, code, err)
	}
	if !grabbed.Results[0].Selected || grabbed.Results[1].Message != "未找到课程" {
		t.Fatalf("unexpected grab results: %#v", grabbed.Results)
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

func TestListCourses_SelectionClosed(t *testing.T) {
	var selectionRequests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/module/module/10/":
			_, _ = w.Write([]byte("AA home"))
		case "/course_selection/courseselecttask/selects/":
			selectionRequests.Add(1)
			_, _ = w.Write([]byte("选课已关闭"))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	l := testCourseRegistrationLogic(t, server.URL, &fakeCourseRegistrationCredentialsRepo{})
	l.sessions[7] = server.Client()

	listed, code, err := l.ListCourses(context.Background(), 7)
	if err != nil || code != httpx.CodeSuccess {
		t.Fatalf("list result = %#v, code = %d, err = %v", listed, code, err)
	}
	if listed.SelectionOpen || listed.Courses == nil || len(listed.Courses) != 0 {
		t.Fatalf("closed selection result = %#v", listed)
	}
	if selectionRequests.Load() != 1 {
		t.Fatalf("selection page requests = %d, want 1", selectionRequests.Load())
	}

	_, code, err = l.GrabCourses(context.Background(), 7, []string{"高等数学"})
	if err == nil || code != httpx.CodeValidationFailed {
		t.Fatalf("grab while closed: code = %d, err = %v", code, err)
	}
}

func TestCourseRegistrationLogin_LoginPageChanged(t *testing.T) {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/home/":
			http.Redirect(w, r, server.URL+"/o/authorize/", http.StatusFound)
		case "/o/authorize/":
			_, _ = w.Write([]byte(`<form id="loginForm"></form>`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	repo := &fakeCourseRegistrationCredentialsRepo{}
	l := testCourseRegistrationLogic(t, server.URL, repo)
	_, code, err := l.Login(context.Background(), 7, "student", "password", "user", "password")
	if err == nil {
		t.Fatal("Login error = nil, want login page error")
	}
	if code != httpx.CodeInternalError {
		t.Fatalf("code = %d, want %d", code, httpx.CodeInternalError)
	}
	if repo.upsertCount != 0 {
		t.Fatalf("failed credentials were saved: upsert count = %d", repo.upsertCount)
	}
}

func TestCourseRegistrationLogin_NoSavedCredentials(t *testing.T) {
	l := NewCourseRegistrationLogic(
		&fakeCourseRegistrationCredentialsRepo{},
		nil,
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
		nil,
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

func TestInspectCourseTargetRequiresExactFullName(t *testing.T) {
	courses := []CourseResult{
		{
			ID: "course-01", Name: "M302006B:计算机体系结构 01 计算机学院",
			Teacher: "田艺", Remaining: 1,
		},
		{
			ID: "course-04", Name: "M302006B:计算机体系结构 04 计算机学院",
			Teacher: "李夏青", Remaining: 1,
		},
	}

	result, state := inspectCourseTarget(courses, "M302006B:计算机体系结构 04 计算机学院")
	if state != targetAvailable || result.Course == nil || result.Course.ID != "course-04" {
		t.Fatalf("exact course match = %#v, state = %d", result, state)
	}

	result, state = inspectCourseTarget(courses, "M302006B")
	if state != targetNotFound || result.Message != "未找到课程" {
		t.Fatalf("short course number unexpectedly matched: %#v, state = %d", result, state)
	}
}

func TestInspectCourseTargetRejectsAmbiguousName(t *testing.T) {
	courses := []CourseResult{
		{ID: "course-a", Name: "M302006B:计算机体系结构 01 计算机学院"},
		{ID: "course-b", Name: "M302006B:计算机体系结构 01 计算机学院"},
	}
	result, state := inspectCourseTarget(courses, "M302006B:计算机体系结构 01 计算机学院")
	if state != targetAmbiguous || result.Message != "课程标识不唯一" {
		t.Fatalf("ambiguous course result = %#v, state = %d", result, state)
	}
}

func TestParseSSOLoginForm_RequiresStableFields(t *testing.T) {
	_, err := parseSSOLoginForm(strings.NewReader(`<form id="loginForm"><input name="flowId" value="flow"></form>`))
	if err == nil {
		t.Fatal("parseSSOLoginForm error = nil")
	}
}

func TestParseSSOLoginForm_UnescapesJSONPublicKey(t *testing.T) {
	form, err := parseSSOLoginForm(strings.NewReader(`<form id="loginForm" action="/sso/login">
<input name="loginType" value="username_password"><input name="flowId" value="flow-id">
<input name="client_id" value="mis-client"><input name="redirect_uri" value="/callback">
<input name="response_type" value="code"></form>
<script>var ssoConfig = {"sm2":{"enabled":true,"publicKey":"\/w=="}};</script>`))
	if err != nil {
		t.Fatal(err)
	}
	if form.publicKey != "/w==" {
		t.Fatalf("public key = %q, want %q", form.publicKey, "/w==")
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
	l := NewCourseRegistrationLogic(repo, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	l.misHomeURL = serverURL + "/home/"
	l.misCourseModuleURL = serverURL + "/module/module/10/"
	l.aaCourseSelectionURL = serverURL + "/course_selection/courseselecttask/selects/"
	l.aaSelectSubmitURL = serverURL + "/course_selection/courseselecttask/selects_action/?action=submit"
	l.aaCaptchaRefreshURL = serverURL + "/captcha/refresh/"
	l.aaCaptchaImageBase = serverURL + "/captcha/image/"
	l.ssoHost = parsed.Host
	l.aaHost = parsed.Host
	l.tujianURL = serverURL + "/predict"
	l.requestTimeout = 2 * time.Second
	return l
}

func testSSOKey(t *testing.T) (*sm2.PrivateKey, string) {
	t.Helper()
	privateKey, err := sm2.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	publicKey := append(
		[]byte{4},
		privateKey.X.FillBytes(make([]byte, 32))...,
	)
	publicKey = append(
		publicKey,
		privateKey.Y.FillBytes(make([]byte, 32))...,
	)
	return privateKey, base64.StdEncoding.EncodeToString(publicKey)
}
