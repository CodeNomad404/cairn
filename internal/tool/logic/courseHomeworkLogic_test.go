package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
)

type homeworkCredentials struct {
	mu    sync.Mutex
	users map[int64]toolmodel.CourseRegistrationCredentials
}

func (r *homeworkCredentials) GetByUserID(_ context.Context, id int64) (*toolmodel.CourseRegistrationCredentials, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	c, ok := r.users[id]
	if !ok {
		return nil, tool.ErrCourseRegistrationCredentialsNotFound
	}
	return &c, nil
}
func (r *homeworkCredentials) Upsert(_ context.Context, c *toolmodel.CourseRegistrationCredentials) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.users[c.UserID] = *c
	return nil
}
func (r *homeworkCredentials) BindMIS(ctx context.Context, c *toolmodel.CourseRegistrationCredentials) error {
	old, err := r.GetByUserID(ctx, c.UserID)
	if err == nil {
		c.TujianUsername = old.TujianUsername
		c.TujianPassword = old.TujianPassword
		c.BindingVersion = old.BindingVersion
	}
	c.BindingVersion++
	return r.Upsert(ctx, c)
}

type memoryHomeworkRepo struct {
	mu          sync.Mutex
	snapshots   map[int64]toolmodel.HomeworkSnapshot
	credentials *homeworkCredentials
	nextID      int64
}

func (r *memoryHomeworkRepo) Snapshot(_ context.Context, u int64) (toolmodel.HomeworkSnapshot, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.snapshots[u], nil
}
func (r *memoryHomeworkRepo) Get(_ context.Context, u, id int64) (*toolmodel.Homework, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, x := range r.snapshots[u].Items {
		if x.ID == id {
			return &x, nil
		}
	}
	return nil, tool.ErrHomeworkNotFound
}
func (r *memoryHomeworkRepo) Prepare(_ context.Context, u, v int64, term string, cs []toolmodel.HomeworkCourse) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	s := r.snapshots[u]
	s.TermCode = term
	s.Courses = append([]toolmodel.HomeworkCourse{}, cs...)
	s.HasSnapshot = true
	r.snapshots[u] = s
	return nil
}
func (r *memoryHomeworkRepo) SaveCourse(ctx context.Context, u, v int64, term string, c toolmodel.HomeworkCourse, items []toolmodel.Homework, message string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	creds, _ := r.credentials.GetByUserID(ctx, u)
	if creds.BindingVersion != v {
		return tool.ErrMISBindingChanged
	}
	s := r.snapshots[u]
	now := time.Now()
	if c.ID == "" {
		if message == "" {
			s.LastSuccessAt = &now
		}
	} else {
		for i := range s.Courses {
			if s.Courses[i].ID == c.ID {
				s.Courses[i].Error = message
				s.Courses[i].LastAttemptAt = now
				if message == "" {
					s.Courses[i].LastSuccessAt = &now
				}
			}
		}
		if message == "" {
			kept := []toolmodel.Homework{}
			for _, x := range s.Items {
				if x.CourseID != c.ID {
					kept = append(kept, x)
				}
			}
			for _, x := range items {
				r.nextID++
				x.ID = r.nextID
				kept = append(kept, x)
			}
			s.Items = kept
		}
	}
	r.snapshots[u] = s
	return nil
}

type noGrabSideEffects struct{ tool.CourseGrabRepo }

func (noGrabSideEffects) RecordLoginAttempt(context.Context, int64) error {
	panic("MIS binding must not touch course-grab state")
}

func homeworkFixture(t *testing.T) (*HomeworkLogic, *memoryHomeworkRepo, *atomic.Int32) {
	t.Helper()
	_, key := testSSOKey(t)
	creds := &homeworkCredentials{users: map[int64]toolmodel.CourseRegistrationCredentials{7: {UserID: 7, StudentID: "student-seven", BJTUPassword: "secret", TujianUsername: "keep", TujianPassword: "also-keep", BindingVersion: 1}, 8: {UserID: 8, StudentID: "student-eight", BJTUPassword: "secret", BindingVersion: 1}}}
	repo := &memoryHomeworkRepo{snapshots: make(map[int64]toolmodel.HomeworkSnapshot), credentials: creds}
	var loginCount atomic.Int32
	var expired atomic.Bool
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/home/":
			if _, err := r.Cookie("MIS"); err != nil {
				http.Redirect(w, r, "/authorize", http.StatusFound)
				return
			}
			_, _ = fmt.Fprint(w, "home")
		case "/authorize":
			_, _ = fmt.Fprintf(w, `<form id="loginForm" action="/login"><input name="username"><input name="password"><input name="loginType" value="username_password"><input name="flowId" value="flow"><input name="client_id" value="mis"><input name="redirect_uri" value="/home/"><input name="response_type" value="code"><input name="state" value="state"></form><script>var ssoConfig={"sm2":{"publicKey":"%s"}}</script>`, key)
		case "/login":
			_ = r.ParseForm()
			if r.Form.Get("username") == "reject" {
				_, _ = fmt.Fprint(w, "rejected")
				return
			}
			loginCount.Add(1)
			http.SetCookie(w, &http.Cookie{Name: "MIS", Value: r.Form.Get("username"), Path: "/"})
			http.Redirect(w, r, "/home/", http.StatusFound)
		case "/module/module/28/":
			_, _ = fmt.Fprintf(w, `<!-- <script>location.href="%s/obsolete"</script> --><script>location.href="%s/ve/index";</script>`, server.URL, server.URL)
		case "/ve/index":
			http.Redirect(w, r, "/ve/back/coursePlatform/coursePlatform.shtml", http.StatusFound)
		case "/ve/back/coursePlatform/coursePlatform.shtml":
			_, _ = fmt.Fprint(w, `<script>function logout(){location.href='/logout'}</script><script>setRequestHeader("sessionId", "fresh-token")</script>`)
		default:
			if r.Header.Get("sessionId") != "fresh-token" {
				http.Error(w, "missing token", http.StatusUnauthorized)
				return
			}
			w.Header().Set("Content-Type", "text/json")
			switch r.URL.Query().Get("method") {
			case "queryCurrentXq":
				_, _ = fmt.Fprint(w, `{"STATUS":"0","result":[{"xqCode":"2026202701","currentFlag":2}]}`)
			case "getCourseList":
				if !expired.Swap(true) {
					http.Redirect(w, r, "/ve/Timeout.jsp", http.StatusFound)
					return
				}
				if r.URL.Query().Get("page") == "1" {
					_, _ = fmt.Fprint(w, `{"STATUS":"0","totalPage":2,"courseList":[{"id":101,"name":"课程一"},{"id":102,"name":"空课程"}]}`)
				} else {
					_, _ = fmt.Fprint(w, `{"STATUS":"0","totalPage":2,"courseList":[{"id":103,"name":"失败课程"}]}`)
				}
			case "getHomeWorkList":
				switch r.URL.Query().Get("cId") {
				case "102":
					_, _ = fmt.Fprint(w, `{"STATUS":"2","totalPage":0,"courseNoteList":[]}`)
				case "103":
					if r.URL.Query().Get("page") == "1" {
						_, _ = fmt.Fprint(w, `{"STATUS":"0","totalPage":2,"courseNoteList":[{"id":998,"title":"不能部分保存的作业","subStatus":"未提交"}]}`)
					} else {
						http.Error(w, "upstream failure", http.StatusBadGateway)
					}
				default:
					id, _ := strconv.Atoi(r.URL.Query().Get("page"))
					_ = json.NewEncoder(w).Encode(map[string]any{"STATUS": "0", "totalPage": 2, "courseNoteList": []any{map[string]any{"id": id, "title": fmt.Sprint("作业", id), "content": "<p>正文</p><script>unsafe()</script>", "user_id": 3, "open_date": "2026-09-01 00:00", "end_time": "2026-12-01 00:00", "subStatus": "未提交"}}})
				}
			case "queryStudentCourseNote":
				_, _ = fmt.Fprint(w, `{"STATUS":"0","homeWork":{"id":1,"content":"<p>详情</p><script>bad()</script>"},"picList":[{"file_name":"作业.pdf"}]}`)
			default:
				http.Error(w, "unexpected action", http.StatusNotFound)
			}
		}
	}))
	t.Cleanup(server.Close)
	mis := testCourseRegistrationLogic(t, server.URL, creds)
	mis.grabRepo = noGrabSideEffects{}
	l := NewHomeworkLogic(repo, mis)
	l.moduleURL = server.URL + "/module/module/28/"
	l.platformURL = server.URL + "/ve/"
	return l, repo, &loginCount
}

func TestHomeworkSyncPaginationPartialFailureAndSessionRenewal(t *testing.T) {
	l, repo, logins := homeworkFixture(t)
	repo.snapshots[7] = toolmodel.HomeworkSnapshot{Items: []toolmodel.Homework{{ID: 90, UserID: 7, BindingVersion: 1, CourseID: "103", Title: "保留旧作业"}}}
	data, code, err := l.Synchronize(context.Background(), 7)
	if err != nil || code != httpx.CodeSuccess {
		t.Fatalf("sync failed: %v", err)
	}
	r := data.(HomeworkSyncResult)
	if r.Status != "partial" || r.Succeeded != 2 || r.Failed != 1 {
		t.Fatalf("result=%+v", r)
	}
	if logins.Load() != 2 {
		t.Fatalf("login count=%d, want original plus one retry", logins.Load())
	}
	s, _ := repo.Snapshot(context.Background(), 7)
	if len(s.Items) != 3 {
		t.Fatalf("items=%+v", s.Items)
	}
	for _, item := range s.Items {
		if item.Content != "" && item.Content != "正文" {
			t.Fatalf("unsafe content=%q", item.Content)
		}
	}
	if s.Items[0].Title != "保留旧作业" {
		t.Fatal("failed course snapshot was replaced")
	}
	data, _, err = l.Get(context.Background(), 7, s.Items[1].ID)
	if err != nil {
		t.Fatal(err)
	}
	d := data.(HomeworkDetailResult)
	if d.Content != "详情" || len(d.Attachments) != 1 {
		t.Fatalf("detail=%+v", d)
	}
	if _, code, err = l.Get(context.Background(), 8, s.Items[1].ID); code != httpx.CodeNotFound || err == nil {
		t.Fatal("another user could read homework")
	}
}

func TestMISBindPreservesTujianAndRejectsWithoutOverwriting(t *testing.T) {
	l, repo, _ := homeworkFixture(t)
	data, _, err := l.mis.BindMIS(context.Background(), 7, "new-student", "new-secret")
	if err != nil {
		t.Fatal(err)
	}
	if !data.(MISBindResult).Bound {
		t.Fatal("not bound")
	}
	c, _ := repo.credentials.GetByUserID(context.Background(), 7)
	if c.StudentID != "new-student" || c.TujianUsername != "keep" || c.TujianPassword != "also-keep" || c.BindingVersion != 2 {
		t.Fatalf("binding not preserved: version=%d", c.BindingVersion)
	}
	if _, _, err = l.mis.BindMIS(context.Background(), 7, "reject", "bad"); err == nil {
		t.Fatal("rejected credentials accepted")
	}
	c, _ = repo.credentials.GetByUserID(context.Background(), 7)
	if c.StudentID != "new-student" {
		t.Fatal("failed binding overwrote valid account")
	}
}

func TestHomeworkClassificationAndPlainText(t *testing.T) {
	now := time.Date(2026, 10, 25, 0, 0, 0, 0, classroomLocation)
	before := now.Add(-time.Second)
	after := now.Add(time.Second)
	for _, tc := range []struct {
		name, submission string
		opens, deadline  *time.Time
		want             string
	}{
		{"pending", "未提交", nil, &after, "pending"}, {"deadline exact", "未提交", nil, &now, "overdue"}, {"returned", "被打回", nil, &before, "overdue"}, {"upcoming", "未提交", &after, &after, "upcoming"}, {"submitted", "已提交", nil, &before, "submitted"}, {"unknown", "new state", nil, nil, "unknown"}, {"no deadline", "未提交", nil, nil, "pending"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			item := homeworkItem(toolmodel.Homework{Submission: tc.submission, OpensAt: tc.opens, Deadline: tc.deadline}, now)
			if item.Status != tc.want {
				t.Fatalf("status=%s want %s", item.Status, tc.want)
			}
		})
	}
	tm, err := homeworkTime("2026-10-26 00:00")
	if err != nil || !tm.Equal(now.AddDate(0, 0, 1)) {
		t.Fatal("midnight parsed incorrectly")
	}
	if _, err = homeworkTime("invalid"); err == nil {
		t.Fatal("invalid timestamp accepted")
	}
	if got := homeworkText(`<p>第一段 &amp; 内容</p><p>第二段<br>换行</p><script>alert(1)</script><style>x</style>`); got != "第一段 & 内容\n第二段\n换行" {
		t.Fatalf("text=%q", got)
	}
}

func TestHomeworkListFiltersCountsAndPagination(t *testing.T) {
	credentials := &homeworkCredentials{users: map[int64]toolmodel.CourseRegistrationCredentials{7: {UserID: 7, StudentID: "24281234"}}}
	before, after := time.Now().Add(-time.Hour), time.Now().Add(time.Hour)
	repo := &memoryHomeworkRepo{credentials: credentials, snapshots: map[int64]toolmodel.HomeworkSnapshot{7: {Items: []toolmodel.Homework{
		{ID: 1, CourseID: "one", Submission: "未提交", Deadline: &after},
		{ID: 2, CourseID: "one", Submission: "未提交", Deadline: &before},
		{ID: 3, CourseID: "two", Submission: "已提交", Deadline: &before},
		{ID: 4, CourseID: "two", Submission: "未提交", OpensAt: &after},
	}}}}
	l := NewHomeworkLogic(repo, NewCourseRegistrationLogic(credentials, nil, nil))
	for page, wantID := range map[int]int64{1: 1, 2: 2} {
		data, _, err := l.List(context.Background(), 7, "", "todo", page, 1)
		if err != nil {
			t.Fatal(err)
		}
		r := data.(HomeworkListResult)
		if !r.Bound || r.StudentID != "24****34" {
			t.Fatalf("binding display=%+v", r)
		}
		if r.Total != 2 || len(r.Items) != 1 || r.Items[0].ID != wantID || r.Counts["submitted"] != 1 || r.Counts["upcoming"] != 1 {
			t.Fatalf("result=%+v", r)
		}
	}
	data, _, _ := l.List(context.Background(), 7, "two", "all", 1, 20)
	if r := data.(HomeworkListResult); r.Total != 2 || r.Counts["pending"] != 1 {
		t.Fatalf("course filter result=%+v", r)
	}
	data, _, _ = l.List(context.Background(), 7, "", "all", 1<<30, 100)
	if len(data.(HomeworkListResult).Items) != 0 {
		t.Fatal("out-of-range page is not empty")
	}
	data, _, err := l.List(context.Background(), 8, "", "all", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if r := data.(HomeworkListResult); r.Bound || r.StudentID != "" {
		t.Fatalf("unbound display=%+v", r)
	}
}

func TestTujianOnlyLoginUsesSavedMISBinding(t *testing.T) {
	l, repo, _ := homeworkFixture(t)
	l.mis.grabRepo = nil
	if _, _, err := l.mis.Login(context.Background(), 7, "", "", "new-tujian", "new-tujian-secret"); err != nil {
		t.Fatal(err)
	}
	c, _ := repo.credentials.GetByUserID(context.Background(), 7)
	if c.StudentID != "student-seven" || c.BJTUPassword != "secret" || c.TujianUsername != "new-tujian" || c.TujianPassword != "new-tujian-secret" {
		t.Fatal("Tujian-only login did not preserve MIS binding")
	}
}

func TestHomeworkUserSessionIsolationAndMissingBinding(t *testing.T) {
	l, _, logins := homeworkFixture(t)
	for _, id := range []int64{7, 8} {
		if _, _, err := l.Synchronize(context.Background(), id); err != nil {
			t.Fatal(err)
		}
	}
	if l.sessions[7].client == l.sessions[8].client || l.sessions[7].client.Jar == l.sessions[8].client.Jar {
		t.Fatal("users share platform session")
	}
	count := logins.Load()
	if _, _, err := l.Synchronize(context.Background(), 8); err != nil {
		t.Fatal(err)
	}
	if logins.Load() != count {
		t.Fatal("valid session was not reused")
	}
	if _, code, err := l.Synchronize(context.Background(), 99); code != httpx.CodeNotFound || !errors.As(err, new(*httpx.AppError)) {
		t.Fatal("missing binding must not fall back to another account")
	}
}
