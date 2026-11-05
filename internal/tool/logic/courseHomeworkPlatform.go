package logic

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"

	toolmodel "cairn.cn/internal/tool/model"
	"cairn.cn/internal/util"
	"golang.org/x/net/html"
)

const defaultCoursePlatformURL = "http://123.121.147.7:88/ve/"

var (
	errPlatformSession = errors.New("课程平台会话已过期")
	platformRedirect   = regexp.MustCompile(`(?:window\.)?location(?:\.href)?\s*=\s*["']([^"']+)["']`)
	platformToken      = regexp.MustCompile(`setRequestHeader\(\s*["']sessionId["']\s*,\s*["']([^"']+)["']`)
)

type platformSession struct {
	gate    chan struct{}
	client  *http.Client
	token   string
	version int64
	retried bool
}

type platformResponse struct {
	Status    string `json:"STATUS"`
	TotalPage int    `json:"totalPage"`
	Terms     []struct {
		Code    string `json:"xqCode"`
		Current int    `json:"currentFlag"`
	} `json:"result"`
	Courses []struct {
		ID     int64  `json:"id"`
		Name   string `json:"name"`
		Number string `json:"course_num"`
		Group  string `json:"fz_id"`
	} `json:"courseList"`
	Homework    []platformHomework `json:"courseNoteList"`
	Detail      *platformHomework  `json:"homeWork"`
	Attachments []struct {
		Name string `json:"file_name"`
	} `json:"picList"`
}

type platformHomework struct {
	ID         int64   `json:"id"`
	TeacherID  int64   `json:"user_id"`
	Title      string  `json:"title"`
	Content    string  `json:"content"`
	Opens      string  `json:"open_date"`
	Deadline   string  `json:"end_time"`
	Submitted  *string `json:"subTime"`
	Submission string  `json:"subStatus"`
	Grading    string  `json:"lastScore"`
}

func (l *HomeworkLogic) platform(ctx context.Context, userID int64) (*platformSession, error) {
	l.mu.Lock()
	p := l.sessions[userID]
	if p == nil {
		p = &platformSession{gate: make(chan struct{}, 1)}
		l.sessions[userID] = p
	}
	l.mu.Unlock()
	select {
	case p.gate <- struct{}{}:
		p.retried = false
		return p, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

func (l *HomeworkLogic) openPlatform(ctx context.Context, p *platformSession, c *toolmodel.CourseRegistrationCredentials) error {
	client, _, err := l.mis.authenticateMIS(ctx, c)
	if err != nil {
		return err
	}
	base, _ := url.Parse(l.platformURL)
	mis, _ := url.Parse(l.mis.misHomeURL)
	client.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if len(via) >= 10 || via[0].Header.Get("sessionId") != "" {
			return http.ErrUseLastResponse
		}
		if req.URL.Host != base.Host && req.URL.Host != mis.Host && req.URL.Host != l.mis.ssoHost {
			return http.ErrUseLastResponse
		}
		return nil
	}
	body, final, err := getPage(ctx, client, l.moduleURL, maxHTMLBytes)
	if err != nil {
		return errors.New("无法进入课程平台")
	}
	for hop := 0; hop < 4; hop++ {
		if final.Host == base.Host && final.Path == base.Path+"back/coursePlatform/coursePlatform.shtml" {
			m := platformToken.FindSubmatch(body)
			if len(m) != 2 {
				return errors.New("课程平台会话格式已变化")
			}
			p.client = client
			p.token = string(m[1])
			p.version = c.BindingVersion
			return nil
		}
		// Parse only executable script nodes: HTML comments contain obsolete entry URLs.
		doc, err := html.Parse(bytes.NewReader(body))
		if err != nil {
			return errors.New("课程平台入口已变化")
		}
		var target string
		walkElements(doc, func(n *html.Node) {
			if target != "" || n.Data != "script" {
				return
			}
			for child := n.FirstChild; child != nil; child = child.NextSibling {
				if child.Type == html.TextNode {
					if m := platformRedirect.FindStringSubmatch(child.Data); len(m) == 2 {
						target = m[1]
						return
					}
				}
			}
		})
		next, err := final.Parse(target)
		if target == "" || err != nil || next.Scheme != base.Scheme || next.Host != base.Host || !strings.HasPrefix(next.Path, base.Path) {
			return errors.New("课程平台入口已变化")
		}
		body, final, err = getPage(ctx, client, next.String(), maxHTMLBytes)
		if err != nil {
			return errors.New("无法打开课程平台")
		}
	}
	return errors.New("课程平台跳转次数过多")
}

func (l *HomeworkLogic) platformAPI(ctx context.Context, p *platformSession, c *toolmodel.CourseRegistrationCredentials, path, action string, q url.Values) (*platformResponse, error) {
	if p.client == nil || p.version != c.BindingVersion {
		if err := l.openPlatform(ctx, p, c); err != nil {
			return nil, err
		}
	}
	q.Set("method", action)
	for {
		v, err := l.platformRequest(ctx, p, l.platformURL+"back/"+path+"?"+q.Encode())
		if errors.Is(err, errPlatformSession) && !p.retried {
			p.retried = true
			p.client = nil
			p.token = ""
			if err = l.openPlatform(ctx, p, c); err != nil {
				return nil, err
			}
			continue
		}
		return v, err
	}
}

func (l *HomeworkLogic) platformRequest(ctx context.Context, p *platformSession, endpoint string) (*platformResponse, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, errors.New("课程平台请求无效")
	}
	req.Header.Set("User-Agent", browserUserAgent)
	req.Header.Set("sessionId", p.token)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	resp, err := p.client.Do(req)
	if err != nil {
		return nil, errors.New("课程平台请求超时或网络不可用")
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode == 302 || resp.StatusCode == 401 || resp.StatusCode == 403 || resp.StatusCode == 500 {
		return nil, errPlatformSession
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("课程平台返回异常状态")
	}
	body, err := util.ReadResponseBody(resp, 4<<20)
	if err != nil {
		return nil, errors.New("课程平台响应过大或读取失败")
	}
	var v platformResponse
	if json.Unmarshal(body, &v) != nil {
		return nil, errors.New("课程平台响应格式已变化")
	}
	if v.Status != "0" && v.Status != "2" {
		return nil, errors.New("课程平台拒绝本次查询")
	}
	return &v, nil
}

func (l *HomeworkLogic) currentCourses(ctx context.Context, p *platformSession, c *toolmodel.CourseRegistrationCredentials) (string, []toolmodel.HomeworkCourse, error) {
	v, err := l.platformAPI(ctx, p, c, "rp/common/teachCalendar.shtml", "queryCurrentXq", url.Values{})
	if err != nil {
		return "", nil, err
	}
	term := ""
	for _, t := range v.Terms {
		if t.Current == 2 {
			term = t.Code
			break
		}
	}
	if term == "" {
		return "", nil, errors.New("未找到当前学期")
	}
	courses := []toolmodel.HomeworkCourse{}
	for page := 1; page <= 100; page++ {
		v, err = l.platformAPI(ctx, p, c, "coursePlatform/course.shtml", "getCourseList", url.Values{"xqCode": {term}, "page": {strconv.Itoa(page)}, "pagesize": {"100"}})
		if err != nil {
			return "", nil, err
		}
		if v.Courses == nil && v.Status != "2" {
			return "", nil, errors.New("课程列表缺少数据字段")
		}
		for _, x := range v.Courses {
			if x.ID <= 0 {
				return "", nil, errors.New("课程标识无效")
			}
			courses = append(courses, toolmodel.HomeworkCourse{ID: strconv.FormatInt(x.ID, 10), Name: x.Name, CourseNumber: x.Number, GroupID: x.Group})
		}
		if v.TotalPage > 100 {
			return "", nil, errors.New("课程分页超过限制，未保存不完整列表")
		}
		if page >= v.TotalPage {
			return term, courses, nil
		}
		if len(v.Courses) == 0 {
			return "", nil, errors.New("课程分页不完整")
		}
	}
	return "", nil, errors.New("课程分页超过限制")
}

func (l *HomeworkLogic) courseHomework(ctx context.Context, p *platformSession, c *toolmodel.CourseRegistrationCredentials, term string, course toolmodel.HomeworkCourse) ([]toolmodel.Homework, error) {
	items := []toolmodel.Homework{}
	for page := 1; page <= 100; page++ {
		v, err := l.platformAPI(ctx, p, c, "coursePlatform/homeWork.shtml", "getHomeWorkList", url.Values{"cId": {course.ID}, "subType": {"0"}, "page": {strconv.Itoa(page)}, "pagesize": {"100"}})
		if err != nil {
			return nil, err
		}
		if v.Homework == nil && v.Status != "2" {
			return nil, errors.New("作业列表缺少数据字段")
		}
		for _, x := range v.Homework {
			if x.ID <= 0 {
				return nil, errors.New("作业标识无效")
			}
			opens, e1 := homeworkTime(x.Opens)
			deadline, e2 := homeworkTime(x.Deadline)
			var sub string
			if x.Submitted != nil {
				sub = *x.Submitted
			}
			submittedAt, e3 := homeworkTime(sub)
			if e1 != nil || e2 != nil || e3 != nil {
				return nil, errors.New("作业时间格式已变化")
			}
			items = append(items, toolmodel.Homework{UserID: c.UserID, BindingVersion: c.BindingVersion, TermCode: term, CourseID: course.ID, CourseName: course.Name, PlatformID: x.ID, TeacherID: x.TeacherID, Title: x.Title, Content: homeworkText(x.Content), OpensAt: opens, Deadline: deadline, SubmittedAt: submittedAt, Submission: x.Submission, Grading: x.Grading})
		}
		if v.TotalPage > 100 {
			return nil, errors.New("作业分页超过限制，未替换旧快照")
		}
		if page >= v.TotalPage {
			return items, nil
		}
		if len(v.Homework) == 0 {
			return nil, errors.New("作业分页不完整")
		}
	}
	return nil, fmt.Errorf("作业分页超过限制")
}
