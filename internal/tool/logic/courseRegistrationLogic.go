package logic

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/emmansun/gmsm/sm2"
	"golang.org/x/net/html"

	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
	"cairn.cn/internal/util"
)

const (
	defaultMISHomeURL           = "https://mis.bjtu.edu.cn/home/"
	defaultMISCourseModuleURL   = "https://mis.bjtu.edu.cn/module/module/10/"
	defaultAACourseSelectionURL = "https://aa.bjtu.edu.cn/course_selection/courseselecttask/selects/"
	defaultAASelectSubmitURL    = "https://aa.bjtu.edu.cn/course_selection/courseselecttask/selects_action/?action=submit"
	defaultAACaptchaRefreshURL  = "https://aa.bjtu.edu.cn/captcha/refresh/"
	defaultAACaptchaImageBase   = "https://aa.bjtu.edu.cn/captcha/image/"
	defaultSSOHost              = "sso.bjtu.edu.cn"
	defaultAAHost               = "aa.bjtu.edu.cn"
	defaultTujianURL            = "https://api.ttshitu.com/predict"
	defaultCoursePollInterval   = 2 * time.Second
	browserUserAgent            = "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/155.0.0.0 Safari/537.36"
	maxHTMLBytes                = 1 << 20
	maxCaptchaBytes             = 2 << 20
)

var ssoPublicKeyPattern = regexp.MustCompile(`"sm2"\s*:\s*\{[^{}]*"publicKey"\s*:\s*"([^"]+)"`)

// CourseRegistrationLogic 负责登录北京交通大学统一身份认证，并按 Cairn 用户保存会话。
type CourseRegistrationLogic struct {
	mu                   sync.RWMutex
	sessions             map[int64]*http.Client
	credentialsRepo      tool.CourseRegistrationCredentialsRepo
	grabRepo             tool.CourseGrabRepo
	logger               *slog.Logger
	misHomeURL           string
	misCourseModuleURL   string
	aaCourseSelectionURL string
	aaSelectSubmitURL    string
	aaCaptchaRefreshURL  string
	aaCaptchaImageBase   string
	ssoHost              string
	aaHost               string
	tujianURL            string
	requestTimeout       time.Duration
	coursePollInterval   time.Duration
	workerMu             sync.Mutex
	workerContext        context.Context
	workers              map[int64]context.CancelFunc
	loginLocks           map[int64]*sync.Mutex
}

type CourseRegistrationLoginResult struct {
	LoggedIn bool `json:"logged_in"`
}

func NewCourseRegistrationLogic(
	credentialsRepo tool.CourseRegistrationCredentialsRepo,
	grabRepo tool.CourseGrabRepo,
	logger *slog.Logger,
) *CourseRegistrationLogic {
	return &CourseRegistrationLogic{
		sessions:             make(map[int64]*http.Client),
		credentialsRepo:      credentialsRepo,
		grabRepo:             grabRepo,
		logger:               logger,
		misHomeURL:           defaultMISHomeURL,
		misCourseModuleURL:   defaultMISCourseModuleURL,
		aaCourseSelectionURL: defaultAACourseSelectionURL,
		aaSelectSubmitURL:    defaultAASelectSubmitURL,
		aaCaptchaRefreshURL:  defaultAACaptchaRefreshURL,
		aaCaptchaImageBase:   defaultAACaptchaImageBase,
		ssoHost:              defaultSSOHost,
		aaHost:               defaultAAHost,
		tujianURL:            defaultTujianURL,
		requestTimeout:       25 * time.Second,
		coursePollInterval:   defaultCoursePollInterval,
		workers:              make(map[int64]context.CancelFunc),
		loginLocks:           make(map[int64]*sync.Mutex),
	}
}

type CourseRegistrationOpenResult struct {
	Entered       bool `json:"entered"`
	SelectionOpen bool `json:"selection_open"`
}

type CourseResult struct {
	ID        string `json:"id,omitempty"`
	Name      string `json:"name"`
	Teacher   string `json:"teacher"`
	Remaining int    `json:"remaining"`
	Selected  bool   `json:"selected"`
}

type CourseListResult struct {
	SelectionOpen bool           `json:"selection_open"`
	Courses       []CourseResult `json:"courses"`
}

type CourseSelectionResult struct {
	Selected bool   `json:"selected"`
	Message  string `json:"message"`
}

type CourseGrabItemResult struct {
	Target   string        `json:"target"`
	TaskID   int64         `json:"task_id,omitempty"`
	Status   string        `json:"status"`
	Course   *CourseResult `json:"course,omitempty"`
	Selected bool          `json:"selected"`
	Message  string        `json:"message"`
}

type CourseGrabResult struct {
	Results []CourseGrabItemResult `json:"results"`
}

type CourseGrabTaskListResult struct {
	WorkerStatus string                     `json:"worker_status"`
	LoginCount   int                        `json:"login_count"`
	LastLoginAt  *time.Time                 `json:"last_login_at,omitempty"`
	Items        []toolmodel.CourseGrabItem `json:"items"`
}

type courseTargetState int

const (
	targetNotFound courseTargetState = iota
	targetNoRemaining
	targetAvailable
	targetSelected
	targetAmbiguous
)

// OpenCourseSelection 沿用登录 Cookie 进入教务系统，并打开“网上选课”页面。
func (l *CourseRegistrationLogic) OpenCourseSelection(
	ctx context.Context,
	userID int64,
) (CourseRegistrationOpenResult, httpx.Code, error) {
	client, code, err := l.openAcademicSystem(ctx, userID)
	if err != nil {
		return CourseRegistrationOpenResult{}, code, err
	}

	// 打开“课程选课 → 选课 → 网上选课”，页面会明确显示当前阶段是否开放。
	body, finalURL, err := getPage(ctx, client, l.aaCourseSelectionURL, maxHTMLBytes)
	if err != nil {
		return CourseRegistrationOpenResult{}, httpx.CodeInternalError,
			httpx.NewAppError("failed to open BJTU course selection", err)
	}
	if finalURL.Host != l.aaHost || strings.HasPrefix(finalURL.Path, "/client/login/") {
		return CourseRegistrationOpenResult{}, httpx.CodeUnauthorized,
			httpx.NewAppError("BJTU academic system session expired", nil)
	}

	selectionOpen := !bytes.Contains(body, []byte("选课已关闭")) &&
		(bytes.Contains(body, []byte(`id="select-submit-btn"`)) || bytes.Contains(body, []byte(`id="current"`)))
	return CourseRegistrationOpenResult{Entered: true, SelectionOpen: selectionOpen}, httpx.CodeSuccess, nil
}

func (l *CourseRegistrationLogic) openAcademicSystem(
	ctx context.Context,
	userID int64,
) (*http.Client, httpx.Code, error) {
	l.mu.RLock()
	client := l.sessions[userID]
	l.mu.RUnlock()
	if client == nil {
		return nil, httpx.CodeUnauthorized,
			httpx.NewAppError("BJTU login session not found", nil)
	}

	// 从 MIS 的“教务系统”模块建立 AA 会话。
	modulePage, finalURL, err := getPage(ctx, client, l.misCourseModuleURL, maxHTMLBytes)
	if err != nil {
		return nil, httpx.CodeInternalError,
			httpx.NewAppError("failed to enter BJTU academic system", err)
	}
	if finalURL.Host != l.aaHost {
		entryURL, err := findURLForHost(bytes.NewReader(modulePage), finalURL, l.aaHost)
		if err != nil {
			return nil, httpx.CodeInternalError,
				httpx.NewAppError("BJTU academic system entry changed", err)
		}
		_, finalURL, err = getPage(ctx, client, entryURL.String(), maxHTMLBytes)
		if err != nil {
			return nil, httpx.CodeInternalError,
				httpx.NewAppError("failed to enter BJTU academic system", err)
		}
		if finalURL.Host != l.aaHost {
			return nil, httpx.CodeUnauthorized,
				httpx.NewAppError("BJTU academic system login failed", nil)
		}
	}
	return client, httpx.CodeSuccess, nil
}

// ListCourses 返回当前用户可见的本方案课程。
func (l *CourseRegistrationLogic) ListCourses(
	ctx context.Context,
	userID int64,
) (CourseListResult, httpx.Code, error) {
	opened, code, err := l.OpenCourseSelection(ctx, userID)
	if err != nil {
		return CourseListResult{}, code, err
	}
	if !opened.SelectionOpen {
		return CourseListResult{Courses: []CourseResult{}}, httpx.CodeSuccess, nil
	}
	client := l.session(userID)
	body, finalURL, err := getPage(ctx, client, l.aaCourseSelectionURL, maxHTMLBytes)
	if err != nil {
		return CourseListResult{}, httpx.CodeInternalError,
			httpx.NewAppError("failed to load BJTU courses", err)
	}
	if finalURL.Host != l.aaHost || strings.HasPrefix(finalURL.Path, "/client/login/") {
		return CourseListResult{}, httpx.CodeUnauthorized,
			httpx.NewAppError("BJTU academic system session expired", nil)
	}
	courses, err := parseCourseList(bytes.NewReader(body))
	if err != nil {
		return CourseListResult{}, httpx.CodeInternalError,
			httpx.NewAppError("BJTU course list changed", err)
	}
	return CourseListResult{SelectionOpen: true, Courses: courses}, httpx.CodeSuccess, nil
}

// GrabCourses 同步确认课程状态，仅将确认暂无余量的课程写入抢课队列。
func (l *CourseRegistrationLogic) GrabCourses(
	ctx context.Context,
	userID int64,
	targets []string,
) (CourseGrabResult, httpx.Code, error) {
	listed, code, err := l.ListCourses(ctx, userID)
	if err != nil {
		return CourseGrabResult{}, code, err
	}
	if !listed.SelectionOpen {
		return CourseGrabResult{}, httpx.CodeValidationFailed,
			httpx.NewAppError("BJTU course selection is closed", nil)
	}
	result := CourseGrabResult{Results: make([]CourseGrabItemResult, 0, len(targets))}
	createdTask := false
	for _, rawTarget := range targets {
		target := normalizeCourseName(rawTarget)
		item, state := inspectCourseTarget(listed.Courses, target)
		switch state {
		case targetNotFound, targetAmbiguous, targetSelected:
			result.Results = append(result.Results, item)
		case targetNoRemaining:
			task := &toolmodel.CourseGrabItem{
				UserID:           userID,
				CourseName:       target,
				UpstreamCourseID: item.Course.ID,
				Teacher:          item.Course.Teacher,
				Remaining:        item.Course.Remaining,
				Status:           toolmodel.CourseGrabStatusGrabbing,
				Message:          "该课程暂无余量，已加入抢课任务",
			}
			if err := l.grabRepo.UpsertItem(ctx, task); err != nil {
				return CourseGrabResult{}, httpx.CodeInternalError,
					httpx.NewAppError("创建抢课任务失败", err)
			}
			item.TaskID = task.ID
			item.Status = task.Status
			item.Message = task.Message
			createdTask = createdTask || task.Status == toolmodel.CourseGrabStatusGrabbing
			result.Results = append(result.Results, item)
		case targetAvailable:
			selected, selectCode, selectErr := l.SelectCourse(ctx, userID, item.Course.ID)
			if selectErr != nil {
				return CourseGrabResult{}, selectCode, selectErr
			}
			item.Selected = selected.Selected
			item.Message = selected.Message
			if item.Selected {
				item.Status = "succeeded"
			} else {
				confirmed, _, confirmErr := l.ListCourses(ctx, userID)
				if confirmErr == nil {
					confirmedItem, confirmedState := inspectCourseTarget(confirmed.Courses, target)
					if confirmedState == targetSelected {
						item = confirmedItem
						item.Status = "succeeded"
						item.Message = "该课程已抢到"
					} else if confirmedState == targetNoRemaining {
						task := &toolmodel.CourseGrabItem{
							UserID:           userID,
							CourseName:       target,
							UpstreamCourseID: confirmedItem.Course.ID,
							Teacher:          confirmedItem.Course.Teacher,
							Remaining:        confirmedItem.Course.Remaining,
							Status:           toolmodel.CourseGrabStatusGrabbing,
							Message:          "该课程暂无余量，已加入抢课任务",
						}
						if err := l.grabRepo.UpsertItem(ctx, task); err != nil {
							return CourseGrabResult{}, httpx.CodeInternalError,
								httpx.NewAppError("创建抢课任务失败", err)
						}
						item = confirmedItem
						item.TaskID = task.ID
						item.Status = task.Status
						item.Message = task.Message
						createdTask = createdTask || task.Status == toolmodel.CourseGrabStatusGrabbing
					}
				}
				if item.Status == "available" {
					item.Status = "unconfirmed"
					if item.Message == "" {
						item.Message = "选课结果未确认"
					}
				}
			}
			result.Results = append(result.Results, item)
		}
	}
	if createdTask {
		if err := l.grabRepo.SetWorkerStatus(ctx, userID, "running"); err != nil {
			return CourseGrabResult{}, httpx.CodeInternalError,
				httpx.NewAppError("启动抢课任务失败", err)
		}
		l.ensureUserWorker(userID)
	}
	return result, httpx.CodeSuccess, nil
}

func inspectCourseTarget(courses []CourseResult, target string) (CourseGrabItemResult, courseTargetState) {
	item := CourseGrabItemResult{Target: target}
	matches := make([]CourseResult, 0, 1)
	for _, course := range courses {
		if strings.EqualFold(normalizeCourseName(course.Name), target) {
			matches = append(matches, course)
		}
	}
	if len(matches) == 0 {
		item.Status = "not_found"
		item.Message = "未找到课程"
		return item, targetNotFound
	}
	if len(matches) > 1 {
		item.Status = "ambiguous"
		item.Message = "课程标识不唯一"
		return item, targetAmbiguous
	}
	item.Course = &matches[0]
	if matches[0].Selected {
		item.Selected = true
		item.Status = "already_selected"
		item.Message = "该课程已选中"
		return item, targetSelected
	}
	if matches[0].ID != "" && matches[0].Remaining > 0 {
		item.Status = "available"
		return item, targetAvailable
	}
	item.Status = "grabbing"
	item.Message = "该课程暂无余量"
	return item, targetNoRemaining
}

func normalizeCourseName(value string) string {
	return strings.Join(strings.Fields(value), " ")
}

func (l *CourseRegistrationLogic) ListCourseGrabTasks(
	ctx context.Context,
	userID int64,
) (CourseGrabTaskListResult, httpx.Code, error) {
	items, err := l.grabRepo.ListItems(ctx, userID)
	if err != nil {
		return CourseGrabTaskListResult{}, httpx.CodeInternalError,
			httpx.NewAppError("读取抢课任务失败", err)
	}
	summary, err := l.grabRepo.GetJobSummary(ctx, userID)
	if err != nil {
		return CourseGrabTaskListResult{}, httpx.CodeInternalError,
			httpx.NewAppError("读取抢课任务失败", err)
	}
	return CourseGrabTaskListResult{
		WorkerStatus: summary.WorkerStatus,
		LoginCount:   summary.LoginCount,
		LastLoginAt:  summary.LastLoginAt,
		Items:        items,
	}, httpx.CodeSuccess, nil
}

func (l *CourseRegistrationLogic) SetCourseGrabTaskStatus(
	ctx context.Context,
	userID int64,
	itemID int64,
	status string,
) (toolmodel.CourseGrabItem, httpx.Code, error) {
	if status != toolmodel.CourseGrabStatusGrabbing && status != toolmodel.CourseGrabStatusPaused {
		return toolmodel.CourseGrabItem{}, httpx.CodeBadRequest,
			httpx.NewAppError("状态只能是 grabbing 或 paused", nil)
	}
	message := "抢课任务已恢复"
	if status == toolmodel.CourseGrabStatusPaused {
		message = "抢课任务已暂停"
	}
	item, err := l.grabRepo.SetItemStatus(ctx, userID, itemID, status, message)
	if errors.Is(err, tool.ErrCourseGrabItemNotFound) {
		return toolmodel.CourseGrabItem{}, httpx.CodeNotFound,
			httpx.NewAppError("未找到可修改的抢课任务", nil)
	}
	if err != nil {
		return toolmodel.CourseGrabItem{}, httpx.CodeInternalError,
			httpx.NewAppError("更新抢课任务失败", err)
	}
	if status == toolmodel.CourseGrabStatusGrabbing {
		if err := l.grabRepo.SetWorkerStatus(ctx, userID, "running"); err != nil {
			return toolmodel.CourseGrabItem{}, httpx.CodeInternalError,
				httpx.NewAppError("恢复抢课任务失败", err)
		}
		l.ensureUserWorker(userID)
	}
	return *item, httpx.CodeSuccess, nil
}

func (l *CourseRegistrationLogic) StartWorkers(ctx context.Context) error {
	l.workerMu.Lock()
	l.workerContext = ctx
	l.workerMu.Unlock()
	userIDs, err := l.grabRepo.ListRunnableUserIDs(ctx)
	if err != nil {
		return err
	}
	for _, userID := range userIDs {
		l.ensureUserWorker(userID)
	}
	return nil
}

func (l *CourseRegistrationLogic) ensureUserWorker(userID int64) {
	l.workerMu.Lock()
	defer l.workerMu.Unlock()
	if _, exists := l.workers[userID]; exists || l.workerContext == nil {
		return
	}
	ctx, cancel := context.WithCancel(l.workerContext)
	l.workers[userID] = cancel
	go l.runUserWorker(ctx, userID)
}

func (l *CourseRegistrationLogic) runUserWorker(ctx context.Context, userID int64) {
	defer func() {
		l.workerMu.Lock()
		delete(l.workers, userID)
		l.workerMu.Unlock()
	}()
	for {
		keepRunning, err := l.processUserCourses(ctx, userID)
		if err != nil {
			l.logger.ErrorContext(ctx, "course grab worker failed", "user_id", userID, "error", err)
		}
		if !keepRunning || ctx.Err() != nil {
			return
		}
		timer := time.NewTimer(l.coursePollInterval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}

func (l *CourseRegistrationLogic) processUserCourses(ctx context.Context, userID int64) (bool, error) {
	items, err := l.grabRepo.ListItemsByStatus(ctx, userID, toolmodel.CourseGrabStatusGrabbing)
	if err != nil {
		return true, err
	}
	if len(items) == 0 {
		return false, l.grabRepo.SetWorkerStatus(ctx, userID, "idle")
	}
	listed, _, err := l.ListCourses(ctx, userID)
	if err != nil && isCourseSessionError(err) {
		if loginErr := l.reloginForWorker(ctx, userID); loginErr != nil {
			_ = l.grabRepo.SetWorkerStatus(ctx, userID, "auth_failed")
			return false, loginErr
		}
		listed, _, err = l.ListCourses(ctx, userID)
	}
	if err != nil {
		return true, err
	}
	if !listed.SelectionOpen {
		return false, l.grabRepo.SetWorkerStatus(ctx, userID, "idle")
	}
	for index := range items {
		if err := l.processCourseGrabItem(ctx, &items[index], listed.Courses); err != nil {
			l.logger.WarnContext(ctx, "course grab item failed", "item_id", items[index].ID, "error", err)
		}
	}
	_ = l.grabRepo.SetWorkerStatus(ctx, userID, "running")
	_ = l.grabRepo.TouchPoll(ctx, userID, time.Now().Add(l.coursePollInterval))
	return true, nil
}

func (l *CourseRegistrationLogic) processCourseGrabItem(
	ctx context.Context,
	item *toolmodel.CourseGrabItem,
	courses []CourseResult,
) error {
	latest, err := l.grabRepo.GetItem(ctx, item.UserID, item.ID)
	if err != nil || latest.Status != toolmodel.CourseGrabStatusGrabbing {
		return err
	}
	result, state := inspectCourseTarget(courses, item.CourseName)
	item.Message = result.Message
	if result.Course != nil {
		item.UpstreamCourseID = result.Course.ID
		item.Teacher = result.Course.Teacher
		item.Remaining = result.Course.Remaining
	}
	switch state {
	case targetNotFound, targetAmbiguous:
		item.Status = toolmodel.CourseGrabStatusNotFound
	case targetSelected:
		item.Status = toolmodel.CourseGrabStatusSucceeded
		item.Message = "该课程已抢到"
	case targetNoRemaining:
		item.Status = toolmodel.CourseGrabStatusGrabbing
		item.Message = "该课程暂无余量，正在抢课"
	case targetAvailable:
		selected, _, selectErr := l.SelectCourse(ctx, item.UserID, result.Course.ID)
		if selectErr != nil {
			return selectErr
		}
		if selected.Selected {
			item.Status = toolmodel.CourseGrabStatusSucceeded
			item.Message = selected.Message
			if item.Message == "" {
				item.Message = "该课程已抢到"
			}
		} else {
			item.Status = toolmodel.CourseGrabStatusGrabbing
			item.Message = selected.Message
			if item.Message == "" {
				item.Message = "选课结果未确认，等待再次检查"
			}
		}
	}
	return l.grabRepo.UpdateItem(ctx, item)
}

func (l *CourseRegistrationLogic) reloginForWorker(ctx context.Context, userID int64) error {
	lock := l.loginLock(userID)
	lock.Lock()
	defer lock.Unlock()
	if _, _, err := l.OpenCourseSelection(ctx, userID); err == nil {
		return nil
	}
	_, _, err := l.Login(ctx, userID, "", "", "", "")
	return err
}

func (l *CourseRegistrationLogic) loginLock(userID int64) *sync.Mutex {
	l.workerMu.Lock()
	defer l.workerMu.Unlock()
	lock := l.loginLocks[userID]
	if lock == nil {
		lock = &sync.Mutex{}
		l.loginLocks[userID] = lock
	}
	return lock
}

func isCourseSessionError(err error) bool {
	var appErr *httpx.AppError
	if !errors.As(err, &appErr) {
		return false
	}
	return appErr.Message == "BJTU login session not found" ||
		appErr.Message == "BJTU academic system session expired"
}

// SelectCourse 识别选课验证码并提交一个课程 ID。
func (l *CourseRegistrationLogic) SelectCourse(
	ctx context.Context,
	userID int64,
	courseID string,
) (CourseSelectionResult, httpx.Code, error) {
	courseID = strings.TrimSpace(courseID)
	if courseID == "" || len(courseID) > 512 {
		return CourseSelectionResult{}, httpx.CodeBadRequest,
			httpx.NewAppError("valid course_id is required", nil)
	}
	if _, code, err := l.OpenCourseSelection(ctx, userID); err != nil {
		return CourseSelectionResult{}, code, err
	}
	client := l.session(userID)

	credentials, err := l.credentialsRepo.GetByUserID(ctx, userID)
	if errors.Is(err, tool.ErrCourseRegistrationCredentialsNotFound) {
		return CourseSelectionResult{}, httpx.CodeNotFound,
			httpx.NewAppError("saved course registration credentials not found", err)
	}
	if err != nil {
		return CourseSelectionResult{}, httpx.CodeInternalError,
			httpx.NewAppError("failed to load course registration credentials", err)
	}

	key, err := l.refreshSelectionCaptcha(ctx, client)
	if err != nil {
		return CourseSelectionResult{}, httpx.CodeInternalError,
			httpx.NewAppError("failed to refresh BJTU selection captcha", err)
	}
	captcha, err := fetchBytes(ctx, client, l.aaCaptchaImageBase+url.PathEscape(key), maxCaptchaBytes)
	if err != nil {
		return CourseSelectionResult{}, httpx.CodeInternalError,
			httpx.NewAppError("failed to fetch BJTU selection captcha", err)
	}
	answer, err := l.solveCaptcha(
		ctx, credentials.TujianUsername, credentials.TujianPassword, 16, captcha,
	)
	if err != nil {
		return CourseSelectionResult{}, httpx.CodeValidationFailed,
			httpx.NewAppError("captcha recognition failed", err)
	}

	values := url.Values{
		"checkboxs": {courseID},
		"hashkey":   {key},
		"answer":    {answer},
	}
	if err := postSelection(ctx, client, l.aaSelectSubmitURL, l.aaCourseSelectionURL, values); err != nil {
		return CourseSelectionResult{}, httpx.CodeInternalError,
			httpx.NewAppError("failed to submit BJTU course selection", err)
	}
	body, finalURL, err := getPage(ctx, client, l.aaCourseSelectionURL, maxHTMLBytes)
	if err != nil {
		return CourseSelectionResult{}, httpx.CodeInternalError,
			httpx.NewAppError("failed to read BJTU course selection result", err)
	}
	if finalURL.Host != l.aaHost || strings.HasPrefix(finalURL.Path, "/client/login/") {
		return CourseSelectionResult{}, httpx.CodeUnauthorized,
			httpx.NewAppError("BJTU academic system session expired", nil)
	}
	message := parseSelectionMessage(body)
	selected := selectionSucceeded(message)
	return CourseSelectionResult{Selected: selected, Message: message}, httpx.CodeSuccess, nil
}

func (l *CourseRegistrationLogic) session(userID int64) *http.Client {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.sessions[userID]
}

func (l *CourseRegistrationLogic) Login(
	ctx context.Context,
	userID int64,
	studentID string,
	password string,
	tujianUsername string,
	tujianPassword string,
) (CourseRegistrationLoginResult, httpx.Code, error) {
	// 1. 使用本次提交的凭据；未提交时读取当前 Cairn 用户已保存的凭据。
	credentials := &toolmodel.CourseRegistrationCredentials{
		UserID:         userID,
		StudentID:      strings.TrimSpace(studentID),
		BJTUPassword:   password,
		TujianUsername: strings.TrimSpace(tujianUsername),
		TujianPassword: tujianPassword,
	}
	shouldSaveCredentials := credentials.StudentID != ""
	if !shouldSaveCredentials {
		var err error
		credentials, err = l.credentialsRepo.GetByUserID(ctx, userID)
		if errors.Is(err, tool.ErrCourseRegistrationCredentialsNotFound) {
			return CourseRegistrationLoginResult{}, httpx.CodeNotFound,
				httpx.NewAppError("saved course registration credentials not found", err)
		}
		if err != nil {
			return CourseRegistrationLoginResult{}, httpx.CodeInternalError,
				httpx.NewAppError("failed to load course registration credentials", err)
		}
	}
	if l.grabRepo != nil {
		if err := l.grabRepo.RecordLoginAttempt(ctx, userID); err != nil {
			return CourseRegistrationLoginResult{}, httpx.CodeInternalError,
				httpx.NewAppError("记录教务系统登录次数失败", err)
		}
	}

	// 2. 创建独立的 Cookie 会话，用于贯穿 MIS、SSO 和后续教务请求。
	client, err := util.NewCookieClient(l.requestTimeout)
	if err != nil {
		return CourseRegistrationLoginResult{}, httpx.CodeInternalError,
			httpx.NewAppError("failed to initialize BJTU login session", err)
	}

	// 3. 从 MIS 进入 SSO，读取 OAuth 登录表单的动态字段和 SM2 公钥。
	loginPage, err := l.openLoginPage(ctx, client)
	if err != nil {
		return CourseRegistrationLoginResult{}, httpx.CodeInternalError,
			httpx.NewAppError("BJTU login service unavailable", err)
	}
	defer func() { _ = loginPage.Body.Close() }()

	form, err := parseSSOLoginForm(io.LimitReader(loginPage.Body, maxHTMLBytes))
	if err != nil {
		return CourseRegistrationLoginResult{}, httpx.CodeInternalError, httpx.NewAppError("BJTU login page changed", err)
	}

	loginURL, err := resolveSSOURL(loginPage.Request.URL, form.action, l.ssoHost)
	if err != nil {
		return CourseRegistrationLoginResult{}, httpx.CodeInternalError, httpx.NewAppError("BJTU login page changed", err)
	}
	encryptedPassword, err := encryptSSOPassword(form.publicKey, credentials.BJTUPassword)
	if err != nil {
		return CourseRegistrationLoginResult{}, httpx.CodeInternalError, httpx.NewAppError("BJTU login page changed", err)
	}

	// 4. 提交 SSO 登录表单；经 OAuth 回调重定向回 MIS 首页即登录成功。
	form.values.Set("username", credentials.StudentID)
	form.values.Set("password", encryptedPassword)
	response, err := postSSOLogin(ctx, client, loginURL, loginPage.Request.URL, form.values)
	if err != nil {
		return CourseRegistrationLoginResult{}, httpx.CodeInternalError, httpx.NewAppError("BJTU login service unavailable", err)
	}
	defer func() { _ = response.Body.Close() }()

	if !samePage(response.Request.URL, l.misHomeURL) {
		return CourseRegistrationLoginResult{}, httpx.CodeUnauthorized, httpx.NewAppError("BJTU login failed", nil)
	}
	// 6. 登录成功后保存新凭据，并保留 Cookie 会话供后续功能复用。
	if shouldSaveCredentials {
		if err := l.credentialsRepo.Upsert(ctx, credentials); err != nil {
			return CourseRegistrationLoginResult{}, httpx.CodeInternalError,
				httpx.NewAppError("failed to save course registration credentials", err)
		}
	}

	l.mu.Lock()
	l.sessions[userID] = client
	l.mu.Unlock()
	if l.grabRepo != nil {
		items, err := l.grabRepo.ListItemsByStatus(ctx, userID, toolmodel.CourseGrabStatusGrabbing)
		if err != nil {
			return CourseRegistrationLoginResult{}, httpx.CodeInternalError,
				httpx.NewAppError("恢复抢课任务失败", err)
		}
		if len(items) > 0 {
			if err := l.grabRepo.SetWorkerStatus(ctx, userID, "running"); err != nil {
				return CourseRegistrationLoginResult{}, httpx.CodeInternalError,
					httpx.NewAppError("恢复抢课任务失败", err)
			}
			l.ensureUserWorker(userID)
		}
	}

	return CourseRegistrationLoginResult{LoggedIn: true}, httpx.CodeSuccess, nil
}

func (l *CourseRegistrationLogic) openLoginPage(ctx context.Context, client *http.Client) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.misHomeURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", browserUserAgent)
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		_ = response.Body.Close()
		return nil, fmt.Errorf("login page returned HTTP %d", response.StatusCode)
	}
	if response.Request.URL.Host != l.ssoHost {
		_ = response.Body.Close()
		return nil, fmt.Errorf("unexpected login host %q", response.Request.URL.Host)
	}
	return response, nil
}

type ssoLoginForm struct {
	action    string
	publicKey string
	values    url.Values
}

func parseSSOLoginForm(r io.Reader) (ssoLoginForm, error) {
	document, err := html.Parse(r)
	if err != nil {
		return ssoLoginForm{}, err
	}

	formNode := findElement(document, "form", "id", "loginForm")
	if formNode == nil {
		return ssoLoginForm{}, fmt.Errorf("login form not found")
	}

	form := ssoLoginForm{values: make(url.Values)}
	form.action = attribute(formNode, "action")
	walkElements(formNode, func(node *html.Node) {
		if node.Data == "input" {
			name := attribute(node, "name")
			if name != "" {
				form.values.Set(name, attribute(node, "value"))
			}
		}
	})
	walkElements(document, func(node *html.Node) {
		if form.publicKey != "" || node.Data != "script" || node.FirstChild == nil {
			return
		}
		match := ssoPublicKeyPattern.FindStringSubmatch(node.FirstChild.Data)
		if len(match) == 2 {
			var publicKey string
			if err := json.Unmarshal([]byte(`"`+match[1]+`"`), &publicKey); err == nil {
				form.publicKey = publicKey
			}
		}
	})

	if form.action == "" {
		return ssoLoginForm{}, fmt.Errorf("login form action not found")
	}
	for _, name := range []string{"loginType", "flowId", "client_id", "redirect_uri", "response_type"} {
		if form.values.Get(name) == "" {
			return ssoLoginForm{}, fmt.Errorf("login field %q not found", name)
		}
	}
	if form.publicKey == "" {
		return ssoLoginForm{}, fmt.Errorf("SM2 public key not found")
	}
	return form, nil
}

func encryptSSOPassword(encodedPublicKey, password string) (string, error) {
	publicKeyBytes, err := base64.StdEncoding.DecodeString(encodedPublicKey)
	if err != nil {
		return "", fmt.Errorf("decode SM2 public key: %w", err)
	}
	if len(publicKeyBytes) == 64 {
		publicKeyBytes = append([]byte{4}, publicKeyBytes...)
	}
	publicKey, err := sm2.NewPublicKey(publicKeyBytes)
	if err != nil {
		return "", fmt.Errorf("parse SM2 public key: %w", err)
	}
	ciphertext, err := sm2.Encrypt(rand.Reader, publicKey, []byte(password), nil)
	if err != nil {
		return "", fmt.Errorf("encrypt password: %w", err)
	}
	if len(ciphertext) < 2 || ciphertext[0] != 4 {
		return "", fmt.Errorf("unexpected SM2 ciphertext")
	}
	return base64.StdEncoding.EncodeToString(ciphertext[1:]), nil
}

func findElement(node *html.Node, element, attributeName, attributeValue string) *html.Node {
	if node.Type == html.ElementNode && node.Data == element && attribute(node, attributeName) == attributeValue {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findElement(child, element, attributeName, attributeValue); found != nil {
			return found
		}
	}
	return nil
}

func walkElements(node *html.Node, visit func(*html.Node)) {
	if node.Type == html.ElementNode {
		visit(node)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		walkElements(child, visit)
	}
}

func attribute(node *html.Node, name string) string {
	for _, attr := range node.Attr {
		if attr.Key == name {
			return attr.Val
		}
	}
	return ""
}

func hasClass(node *html.Node, class string) bool {
	for value := range strings.FieldsSeq(attribute(node, "class")) {
		if value == class {
			return true
		}
	}
	return false
}

func resolveSSOURL(base *url.URL, reference, expectedHost string) (*url.URL, error) {
	resolved, err := base.Parse(reference)
	if err != nil {
		return nil, err
	}
	if resolved.Host != expectedHost {
		return nil, fmt.Errorf("unexpected SSO host %q", resolved.Host)
	}
	return resolved, nil
}

func findURLForHost(r io.Reader, base *url.URL, expectedHost string) (*url.URL, error) {
	document, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	var found *url.URL
	walkElements(document, func(node *html.Node) {
		if found != nil {
			return
		}
		for _, name := range []string{"href", "action", "src"} {
			reference := attribute(node, name)
			if reference == "" {
				continue
			}
			resolved, err := base.Parse(reference)
			if err == nil && resolved.Host == expectedHost {
				found = resolved
				return
			}
		}
	})
	if found == nil {
		return nil, fmt.Errorf("link to host %q not found", expectedHost)
	}
	return found, nil
}

// fetchBytes 从指定的 URL 获取响应体，并限制其大小不超过 maxBytes。
func fetchBytes(ctx context.Context, client *http.Client, endpoint string, maxBytes int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, err
	}
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request returned HTTP %d", response.StatusCode)
	}
	return util.ReadResponseBody(response, maxBytes)
}

func getPage(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	maxBytes int64,
) ([]byte, *url.URL, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("User-Agent", browserUserAgent)
	response, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, response.Request.URL, &httpStatusError{StatusCode: response.StatusCode}
	}
	body, err := util.ReadResponseBody(response, maxBytes)
	return body, response.Request.URL, err
}

type httpStatusError struct {
	StatusCode int
}

func (e *httpStatusError) Error() string {
	return fmt.Sprintf("request returned HTTP %d", e.StatusCode)
}

// solveCaptcha 使用图鉴 API 识别验证码图像，并返回识别结果。
func (l *CourseRegistrationLogic) solveCaptcha(
	ctx context.Context,
	username string,
	password string,
	typeID int,
	image []byte,
) (string, error) {
	payload, err := json.Marshal(struct {
		Username string `json:"username"`
		Password string `json:"password"`
		TypeID   int    `json:"typeid"`
		Image    string `json:"image"`
	}{
		Username: username,
		Password: password,
		TypeID:   typeID,
		Image:    base64.StdEncoding.EncodeToString(image),
	})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, l.tujianURL, bytes.NewReader(payload))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	client := &http.Client{Timeout: l.requestTimeout}
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Tujian returned HTTP %d", response.StatusCode)
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
		Data    struct {
			Result string `json:"result"`
		} `json:"data"`
	}
	decoder := json.NewDecoder(io.LimitReader(response.Body, maxHTMLBytes))
	if err := decoder.Decode(&result); err != nil {
		return "", err
	}
	answer := strings.TrimSpace(result.Data.Result)
	if !result.Success || answer == "" {
		return "", fmt.Errorf("Tujian rejected captcha request: %s", result.Message)
	}
	return answer, nil
}

func (l *CourseRegistrationLogic) refreshSelectionCaptcha(
	ctx context.Context,
	client *http.Client,
) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.aaCaptchaRefreshURL, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Referer", l.aaCourseSelectionURL)
	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return "", fmt.Errorf("captcha refresh returned HTTP %d", response.StatusCode)
	}
	var result struct {
		Key string `json:"key"`
	}
	if err := json.NewDecoder(io.LimitReader(response.Body, maxHTMLBytes)).Decode(&result); err != nil {
		return "", err
	}
	if strings.TrimSpace(result.Key) == "" {
		return "", fmt.Errorf("captcha refresh response has no key")
	}
	return result.Key, nil
}

func postSelection(
	ctx context.Context,
	client *http.Client,
	endpoint string,
	referer string,
	values url.Values,
) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(values.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", "https://aa.bjtu.edu.cn")
	req.Header.Set("Referer", referer)
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	response, err := client.Do(req)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("course selection returned HTTP %d", response.StatusCode)
	}
	_, err = util.ReadResponseBody(response, maxHTMLBytes)
	return err
}

func parseCourseList(r io.Reader) ([]CourseResult, error) {
	document, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	tables := findElements(document, "table")
	tableIndex := 1
	nameColumn, remainingColumn, teacherColumn := 1, 2, 6
	if len(tables) <= tableIndex {
		return []CourseResult{}, nil
	}

	courses := make([]CourseResult, 0)
	for _, row := range findElements(tables[tableIndex], "tr") {
		columns := directChildren(row, "td")
		if len(columns) <= teacherColumn {
			continue
		}
		id := ""
		inputs := findElements(columns[0], "input")
		if len(inputs) > 0 {
			id = strings.TrimSpace(attribute(inputs[0], "value"))
		}
		status := elementText(columns[0])
		remaining, _ := strconv.Atoi(elementText(columns[remainingColumn]))
		courses = append(courses, CourseResult{
			ID:        id,
			Name:      elementText(columns[nameColumn]),
			Teacher:   elementText(columns[teacherColumn]),
			Remaining: remaining,
			Selected:  id == "" && strings.Contains(status, "已选"),
		})
	}
	return courses, nil
}

func findElements(node *html.Node, element string) []*html.Node {
	var result []*html.Node
	if node.Type == html.ElementNode && node.Data == element {
		result = append(result, node)
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		result = append(result, findElements(child, element)...)
	}
	return result
}

func directChildren(node *html.Node, element string) []*html.Node {
	var result []*html.Node
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && child.Data == element {
			result = append(result, child)
		}
	}
	return result
}

func elementText(node *html.Node) string {
	var builder strings.Builder
	var walk func(*html.Node)
	walk = func(current *html.Node) {
		if current.Type == html.TextNode {
			builder.WriteString(current.Data)
		}
		for child := current.FirstChild; child != nil; child = child.NextSibling {
			walk(child)
		}
	}
	walk(node)
	return strings.Join(strings.Fields(builder.String()), " ")
}

var selectionMessagePattern = regexp.MustCompile(`message\s*\+=\s*["'](.*?)<br\s*/?>["']\s*;`)

func parseSelectionMessage(body []byte) string {
	matches := selectionMessagePattern.FindAllSubmatch(body, -1)
	messages := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) == 2 {
			messages = append(messages, strings.TrimSpace(string(match[1])))
		}
	}
	return strings.Join(messages, "; ")
}

func selectionSucceeded(message string) bool {
	if message == "" || strings.Contains(message, "失败") || strings.Contains(message, "不能") ||
		strings.Contains(message, "冲突") || strings.Contains(message, "已满") {
		return false
	}
	return strings.Contains(message, "成功") || strings.Contains(message, "已选")
}

func postSSOLogin(
	ctx context.Context,
	client *http.Client,
	loginURL *url.URL,
	referer *url.URL,
	values url.Values,
) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, loginURL.String(), strings.NewReader(values.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Origin", loginURL.Scheme+"://"+loginURL.Host)
	req.Header.Set("Referer", referer.String())
	req.Header.Set("User-Agent", browserUserAgent)
	return client.Do(req)
}

func samePage(actual *url.URL, expected string) bool {
	expectedURL, err := url.Parse(expected)
	if err != nil {
		return false
	}
	return actual.Scheme == expectedURL.Scheme && actual.Host == expectedURL.Host && actual.Path == expectedURL.Path
}
