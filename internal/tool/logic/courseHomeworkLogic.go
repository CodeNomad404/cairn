package logic

import (
	"context"
	"errors"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
	"golang.org/x/net/html"
)

type HomeworkLogic struct {
	repo        tool.HomeworkRepo
	mis         *CourseRegistrationLogic
	mu          sync.Mutex
	sessions    map[int64]*platformSession
	runs        map[int64]*homeworkRun
	moduleURL   string
	platformURL string
}

type homeworkRun struct {
	done chan struct{}
	data any
	code httpx.Code
	err  error
}

func NewHomeworkLogic(repo tool.HomeworkRepo, mis *CourseRegistrationLogic) *HomeworkLogic {
	return &HomeworkLogic{repo: repo, mis: mis, sessions: make(map[int64]*platformSession), runs: make(map[int64]*homeworkRun), moduleURL: "https://mis.bjtu.edu.cn/module/module/28/", platformURL: defaultCoursePlatformURL}
}

type HomeworkItemResult struct {
	ID          int64      `json:"id"`
	CourseID    string     `json:"course_id"`
	CourseName  string     `json:"course_name"`
	Title       string     `json:"title"`
	OpensAt     *time.Time `json:"opens_at"`
	Deadline    *time.Time `json:"deadline"`
	SubmittedAt *time.Time `json:"submitted_at"`
	Submission  string     `json:"submission"`
	Status      string     `json:"status"`
	Grading     string     `json:"grading"`
}

type HomeworkCourseResult struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	LastAttemptAt time.Time  `json:"last_attempt_at"`
	LastSuccessAt *time.Time `json:"last_success_at"`
	Error         string     `json:"error"`
}

type HomeworkListResult struct {
	Bound         bool                   `json:"bound"`
	StudentID     string                 `json:"student_id"`
	HasSnapshot   bool                   `json:"has_snapshot"`
	TermCode      string                 `json:"term_code"`
	LastSuccessAt *time.Time             `json:"last_success_at"`
	Courses       []HomeworkCourseResult `json:"courses"`
	Items         []HomeworkItemResult   `json:"items"`
	Counts        map[string]int         `json:"counts"`
	Total         int                    `json:"total"`
	Page          int                    `json:"page"`
	PageSize      int                    `json:"page_size"`
}

type HomeworkSyncResult struct {
	Status        string                 `json:"status"`
	Succeeded     int                    `json:"succeeded"`
	Failed        int                    `json:"failed"`
	TermCode      string                 `json:"term_code"`
	LastSuccessAt *time.Time             `json:"last_success_at"`
	Courses       []HomeworkCourseResult `json:"courses"`
}

type HomeworkAttachmentResult struct {
	Name string `json:"name"`
}
type HomeworkDetailResult struct {
	Item        HomeworkItemResult         `json:"item"`
	Content     string                     `json:"content"`
	Attachments []HomeworkAttachmentResult `json:"attachments"`
	Warning     string                     `json:"warning,omitempty"`
}

func homeworkItem(x toolmodel.Homework, now time.Time) HomeworkItemResult {
	status := "unknown"
	switch x.Submission {
	case "已提交":
		status = "submitted"
	case "未提交", "被打回":
		status = "pending"
		if x.OpensAt != nil && x.OpensAt.After(now) {
			status = "upcoming"
		} else if x.Deadline != nil && !x.Deadline.After(now) {
			status = "overdue"
		}
	}
	return HomeworkItemResult{ID: x.ID, CourseID: x.CourseID, CourseName: x.CourseName, Title: x.Title, OpensAt: x.OpensAt, Deadline: x.Deadline, SubmittedAt: x.SubmittedAt, Submission: x.Submission, Status: status, Grading: x.Grading}
}

func homeworkCourses(items []toolmodel.HomeworkCourse) []HomeworkCourseResult {
	result := make([]HomeworkCourseResult, 0, len(items))
	for _, x := range items {
		result = append(result, HomeworkCourseResult{ID: x.ID, Name: x.Name, LastAttemptAt: x.LastAttemptAt, LastSuccessAt: x.LastSuccessAt, Error: x.Error})
	}
	return result
}

func (l *HomeworkLogic) List(ctx context.Context, userID int64, courseID, status string, page, pageSize int) (any, httpx.Code, error) {
	credentials, err := l.mis.credentialsRepo.GetByUserID(ctx, userID)
	bound := err == nil
	if err != nil && !errors.Is(err, tool.ErrCourseRegistrationCredentialsNotFound) {
		return nil, httpx.CodeInternalError, httpx.NewAppError("读取 MIS 绑定失败", err)
	}
	s, err := l.repo.Snapshot(ctx, userID)
	if err != nil {
		return nil, httpx.CodeInternalError, httpx.NewAppError("读取作业快照失败", err)
	}
	result := HomeworkListResult{Bound: bound, HasSnapshot: s.HasSnapshot, TermCode: s.TermCode, LastSuccessAt: s.LastSuccessAt, Courses: homeworkCourses(s.Courses), Items: []HomeworkItemResult{}, Counts: map[string]int{"pending": 0, "overdue": 0, "submitted": 0, "upcoming": 0, "unknown": 0}, Page: page, PageSize: pageSize}
	if bound {
		result.StudentID = maskedMISStudentID(credentials.StudentID)
	}
	items := []HomeworkItemResult{}
	now := time.Now()
	for _, x := range s.Items {
		item := homeworkItem(x, now)
		result.Counts[item.Status]++
		matches := status == "all" || status == item.Status || status == "todo" && (item.Status == "pending" || item.Status == "overdue")
		if matches && (courseID == "" || courseID == item.CourseID) {
			items = append(items, item)
		}
	}
	rank := map[string]int{"pending": 0, "overdue": 1, "submitted": 2, "upcoming": 3, "unknown": 4}
	sort.Slice(items, func(i, j int) bool {
		a, b := items[i], items[j]
		if rank[a.Status] != rank[b.Status] {
			return rank[a.Status] < rank[b.Status]
		}
		if a.Deadline == nil && b.Deadline != nil {
			return false
		}
		if a.Deadline != nil && b.Deadline == nil {
			return true
		}
		if a.Deadline != nil && b.Deadline != nil && !a.Deadline.Equal(*b.Deadline) {
			return a.Deadline.Before(*b.Deadline)
		}
		return a.ID < b.ID
	})
	result.Total = len(items)
	if page <= len(items)/pageSize+1 {
		start := (page - 1) * pageSize
		end := min(start+pageSize, len(items))
		result.Items = items[start:end]
	}
	return result, httpx.CodeSuccess, nil
}

func homeworkFailure(err error) (any, httpx.Code, error) {
	if errors.Is(err, tool.ErrCourseRegistrationCredentialsNotFound) {
		return nil, httpx.CodeNotFound, httpx.NewAppError("请先绑定 MIS 账号", nil)
	}
	if errors.Is(err, tool.ErrMISBindingChanged) {
		return nil, httpx.CodeBadRequest, httpx.NewAppError("MIS 绑定已变化，请重新刷新", nil)
	}
	var app *httpx.AppError
	if errors.As(err, &app) && app.Message == "BJTU login failed" {
		return nil, httpx.CodeUnauthorized, httpx.NewAppError("MIS 登录失败，请检查绑定信息", nil)
	}
	return nil, httpx.CodeInternalError, httpx.NewAppError("同步失败，请稍后重试或检查 MIS 绑定", nil)
}

func (l *HomeworkLogic) Synchronize(ctx context.Context, userID int64) (any, httpx.Code, error) {
	l.mu.Lock()
	if run := l.runs[userID]; run != nil {
		l.mu.Unlock()
		select {
		case <-run.done:
			return run.data, run.code, run.err
		case <-ctx.Done():
			return homeworkFailure(ctx.Err())
		}
	}
	run := &homeworkRun{done: make(chan struct{})}
	l.runs[userID] = run
	l.mu.Unlock()
	defer func() { l.mu.Lock(); delete(l.runs, userID); close(run.done); l.mu.Unlock() }()
	run.data, run.code, run.err = l.synchronize(ctx, userID)
	return run.data, run.code, run.err
}

func (l *HomeworkLogic) synchronize(parent context.Context, userID int64) (any, httpx.Code, error) {
	ctx, cancel := context.WithTimeout(parent, 22*time.Second)
	defer cancel()
	persistCtx, finish := context.WithTimeout(context.WithoutCancel(parent), 26*time.Second)
	defer finish()
	c, err := l.mis.credentialsRepo.GetByUserID(ctx, userID)
	if err != nil {
		return homeworkFailure(err)
	}
	p, err := l.platform(ctx, userID)
	if err != nil {
		return homeworkFailure(err)
	}
	defer func() { <-p.gate }()
	term, courses, err := l.currentCourses(ctx, p, c)
	if err != nil {
		return homeworkFailure(err)
	}
	if err = l.repo.Prepare(ctx, userID, c.BindingVersion, term, courses); err != nil {
		return homeworkFailure(err)
	}
	result := HomeworkSyncResult{Status: "success", TermCode: term}
	for _, course := range courses {
		items, fetchErr := l.courseHomework(ctx, p, c, term, course)
		message := ""
		if fetchErr != nil {
			message = "课程更新失败，请重试"
			if ctx.Err() != nil {
				message = "同步超时，保留上次结果"
			}
		}
		// One overall deadline reserves time for recording failures after upstream timeout.
		err = l.repo.SaveCourse(persistCtx, userID, c.BindingVersion, term, course, items, message)
		if err != nil {
			return homeworkFailure(err)
		}
		if fetchErr == nil {
			result.Succeeded++
		} else {
			result.Failed++
		}
	}
	message := ""
	if result.Failed > 0 {
		result.Status = "partial"
		message = "部分课程同步失败"
		if result.Succeeded == 0 {
			result.Status = "failed"
			message = "课程同步失败"
		}
	}
	err = l.repo.SaveCourse(persistCtx, userID, c.BindingVersion, term, toolmodel.HomeworkCourse{}, nil, message)
	if err != nil {
		return homeworkFailure(err)
	}
	s, err := l.repo.Snapshot(persistCtx, userID)
	if err != nil {
		return homeworkFailure(err)
	}
	current, err := l.mis.credentialsRepo.GetByUserID(persistCtx, userID)
	if err != nil {
		return homeworkFailure(err)
	}
	if current.BindingVersion != c.BindingVersion {
		return homeworkFailure(tool.ErrMISBindingChanged)
	}
	result.Courses = homeworkCourses(s.Courses)
	result.LastSuccessAt = s.LastSuccessAt
	return result, httpx.CodeSuccess, nil
}

func (l *HomeworkLogic) Get(ctx context.Context, userID, id int64) (any, httpx.Code, error) {
	x, err := l.repo.Get(ctx, userID, id)
	if errors.Is(err, tool.ErrHomeworkNotFound) {
		return nil, httpx.CodeNotFound, httpx.NewAppError("未找到作业", nil)
	}
	if err != nil {
		return nil, httpx.CodeInternalError, httpx.NewAppError("读取作业失败", err)
	}
	result := HomeworkDetailResult{Item: homeworkItem(*x, time.Now()), Content: x.Content, Attachments: []HomeworkAttachmentResult{}}
	c, err := l.mis.credentialsRepo.GetByUserID(ctx, userID)
	if err != nil || c.BindingVersion != x.BindingVersion {
		return nil, httpx.CodeNotFound, httpx.NewAppError("绑定已变化，请刷新作业", nil)
	}
	detailCtx, cancel := context.WithTimeout(ctx, 18*time.Second)
	defer cancel()
	p, err := l.platform(detailCtx, userID)
	if err != nil {
		result.Warning = "详情更新失败，显示缓存正文"
		return l.checkedDetail(ctx, userID, x, result)
	}
	defer func() { <-p.gate }()
	v, err := l.platformAPI(detailCtx, p, c, "coursePlatform/homeWork.shtml", "queryStudentCourseNote", url.Values{"id": {strconv.FormatInt(x.PlatformID, 10)}, "courseId": {x.CourseID}, "teacherId": {strconv.FormatInt(x.TeacherID, 10)}})
	if err != nil || v.Detail == nil || v.Detail.ID != x.PlatformID {
		result.Warning = "详情更新失败，显示缓存正文；附件暂不可用"
	} else {
		result.Content = homeworkText(v.Detail.Content)
		for _, a := range v.Attachments {
			result.Attachments = append(result.Attachments, HomeworkAttachmentResult{Name: a.Name})
		}
	}
	// Recheck ownership/version after the network request, including concurrent account changes.
	return l.checkedDetail(ctx, userID, x, result)
}

func (l *HomeworkLogic) checkedDetail(ctx context.Context, userID int64, original *toolmodel.Homework, result HomeworkDetailResult) (any, httpx.Code, error) {
	current, err := l.repo.Get(ctx, userID, original.ID)
	if err != nil || current.BindingVersion != original.BindingVersion {
		return nil, httpx.CodeNotFound, httpx.NewAppError("绑定或作业已变化，请刷新", nil)
	}
	return result, httpx.CodeSuccess, nil
}

func homeworkTime(raw string) (*time.Time, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, nil
	}
	for _, layout := range []string{"2006-01-02 15:04:05", "2006-01-02 15:04", time.DateOnly} {
		if t, err := time.ParseInLocation(layout, raw, classroomLocation); err == nil {
			return &t, nil
		}
	}
	return nil, errors.New("invalid homework timestamp")
}

func homeworkText(raw string) string {
	doc, err := html.Parse(strings.NewReader(raw))
	if err != nil {
		return ""
	}
	var b strings.Builder
	var walk func(*html.Node)
	walk = func(n *html.Node) {
		if n.Type == html.ElementNode && (n.Data == "script" || n.Data == "style") {
			return
		}
		if n.Type == html.TextNode {
			b.WriteString(n.Data)
		}
		if n.Type == html.ElementNode && (n.Data == "br" || n.Data == "p" || n.Data == "div" || n.Data == "li") {
			b.WriteByte('\n')
		}
		for c := n.FirstChild; c != nil; c = c.NextSibling {
			walk(c)
		}
		if n.Type == html.ElementNode && (n.Data == "p" || n.Data == "div" || n.Data == "li") {
			b.WriteByte('\n')
		}
	}
	walk(doc)
	lines := []string{}
	for _, line := range strings.Split(b.String(), "\n") {
		if line = strings.TrimSpace(line); line != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}
