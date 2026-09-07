package logic

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

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
	defaultCASHost              = "cas.bjtu.edu.cn"
	defaultAAHost               = "aa.bjtu.edu.cn"
	defaultTujianURL            = "https://api.ttshitu.com/predict"
	maxHTMLBytes                = 1 << 20
	maxCaptchaBytes             = 2 << 20
)

// CourseRegistrationLogic 负责登录北京交通大学统一身份认证，并按 Cairn 用户保存会话。
type CourseRegistrationLogic struct {
	mu                   sync.RWMutex
	sessions             map[int64]*http.Client
	credentialsRepo      tool.CourseRegistrationCredentialsRepo
	logger               *slog.Logger
	misHomeURL           string
	misCourseModuleURL   string
	aaCourseSelectionURL string
	casHost              string
	aaHost               string
	tujianURL            string
	requestTimeout       time.Duration
}

type CourseRegistrationLoginResult struct {
	LoggedIn bool `json:"logged_in"`
}

func NewCourseRegistrationLogic(
	credentialsRepo tool.CourseRegistrationCredentialsRepo,
	logger *slog.Logger,
) *CourseRegistrationLogic {
	return &CourseRegistrationLogic{
		sessions:             make(map[int64]*http.Client),
		credentialsRepo:      credentialsRepo,
		logger:               logger,
		misHomeURL:           defaultMISHomeURL,
		misCourseModuleURL:   defaultMISCourseModuleURL,
		aaCourseSelectionURL: defaultAACourseSelectionURL,
		casHost:              defaultCASHost,
		aaHost:               defaultAAHost,
		tujianURL:            defaultTujianURL,
		requestTimeout:       25 * time.Second,
	}
}

type CourseRegistrationOpenResult struct {
	Entered       bool `json:"entered"`
	SelectionOpen bool `json:"selection_open"`
}

// OpenCourseSelection 沿用登录 Cookie 进入教务系统，并打开“网上选课”页面。
func (l *CourseRegistrationLogic) OpenCourseSelection(
	ctx context.Context,
	userID int64,
) (CourseRegistrationOpenResult, httpx.Code, error) {
	l.mu.RLock()
	client := l.sessions[userID]
	l.mu.RUnlock()
	if client == nil {
		return CourseRegistrationOpenResult{}, httpx.CodeUnauthorized,
			httpx.NewAppError("BJTU login session not found", nil)
	}

	// 1. 与 CC.py 一致，先从 MIS 的“教务系统”模块建立 AA 会话。
	modulePage, finalURL, err := getPage(ctx, client, l.misCourseModuleURL, maxHTMLBytes)
	if err != nil {
		return CourseRegistrationOpenResult{}, httpx.CodeInternalError,
			httpx.NewAppError("failed to enter BJTU academic system", err)
	}
	if finalURL.Host != l.aaHost {
		entryURL, err := findURLForHost(bytes.NewReader(modulePage), finalURL, l.aaHost)
		if err != nil {
			return CourseRegistrationOpenResult{}, httpx.CodeInternalError,
				httpx.NewAppError("BJTU academic system entry changed", err)
		}
		_, finalURL, err = getPage(ctx, client, entryURL.String(), maxHTMLBytes)
		if err != nil {
			return CourseRegistrationOpenResult{}, httpx.CodeInternalError,
				httpx.NewAppError("failed to enter BJTU academic system", err)
		}
		if finalURL.Host != l.aaHost {
			return CourseRegistrationOpenResult{}, httpx.CodeUnauthorized,
				httpx.NewAppError("BJTU academic system login failed", nil)
		}
	}

	// 2. 打开“课程选课 → 选课 → 网上选课”，页面会明确显示当前阶段是否开放。
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

	// 2. 创建独立的 Cookie 会话，用于贯穿 MIS、CAS 和后续教务请求。
	client, err := util.NewCookieClient(l.requestTimeout)
	if err != nil {
		return CourseRegistrationLoginResult{}, httpx.CodeInternalError,
			httpx.NewAppError("failed to initialize BJTU login session", err)
	}

	// 3. 从 MIS 进入 CAS，读取登录表单中的动态字段和验证码地址。
	loginPage, err := l.openLoginPage(ctx, client)
	if err != nil {
		return CourseRegistrationLoginResult{}, httpx.CodeInternalError,
			httpx.NewAppError("BJTU login service unavailable", err)
	}
	defer loginPage.Body.Close()

	form, err := parseCASLoginForm(io.LimitReader(loginPage.Body, maxHTMLBytes))
	if err != nil {
		return CourseRegistrationLoginResult{}, httpx.CodeInternalError, httpx.NewAppError("BJTU login page changed", err)
	}

	loginURL, err := resolveCASURL(loginPage.Request.URL, form.action, l.casHost)
	if err != nil {
		return CourseRegistrationLoginResult{}, httpx.CodeInternalError, httpx.NewAppError("BJTU login page changed", err)
	}
	captchaURL, err := resolveCASURL(loginPage.Request.URL, form.captchaPath, l.casHost)
	if err != nil {
		return CourseRegistrationLoginResult{}, httpx.CodeInternalError, httpx.NewAppError("BJTU login page changed", err)
	}

	// 4. 下载验证码并交给图鉴识别。
	captcha, err := fetchBytes(ctx, client, captchaURL.String(), maxCaptchaBytes)
	if err != nil {
		return CourseRegistrationLoginResult{}, httpx.CodeInternalError, httpx.NewAppError("failed to fetch BJTU captcha", err)
	}
	captchaAnswer, err := l.solveCaptcha(ctx, credentials.TujianUsername, credentials.TujianPassword, captcha)
	if err != nil {
		l.logger.WarnContext(ctx, "course registration captcha recognition failed", "error", err)
		return CourseRegistrationLoginResult{}, httpx.CodeValidationFailed, httpx.NewAppError("captcha recognition failed", err)
	}

	// 5. 提交 CAS 登录表单；重定向回 MIS 首页即登录成功。
	form.values.Set("loginname", credentials.StudentID)
	form.values.Set("password", credentials.BJTUPassword)
	form.values.Set("captcha_1", captchaAnswer)
	response, err := postCASLogin(ctx, client, loginURL, loginPage.Request.URL, form.values)
	if err != nil {
		return CourseRegistrationLoginResult{}, httpx.CodeInternalError, httpx.NewAppError("BJTU login service unavailable", err)
	}
	defer response.Body.Close()

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

	return CourseRegistrationLoginResult{LoggedIn: true}, httpx.CodeSuccess, nil
}
func (l *CourseRegistrationLogic) openLoginPage(ctx context.Context, client *http.Client) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.misHomeURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Cairn/1.0")
	response, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if response.StatusCode != http.StatusOK {
		response.Body.Close()
		return nil, fmt.Errorf("login page returned HTTP %d", response.StatusCode)
	}
	if response.Request.URL.Host != l.casHost {
		response.Body.Close()
		return nil, fmt.Errorf("unexpected login host %q", response.Request.URL.Host)
	}
	return response, nil
}

type casLoginForm struct {
	action      string
	captchaPath string
	values      url.Values
}

func parseCASLoginForm(r io.Reader) (casLoginForm, error) {
	document, err := html.Parse(r)
	if err != nil {
		return casLoginForm{}, err
	}

	formNode := findElement(document, "form", "id", "login")
	if formNode == nil {
		return casLoginForm{}, fmt.Errorf("login form not found")
	}

	form := casLoginForm{values: make(url.Values)}
	form.action = attribute(formNode, "action")
	walkElements(formNode, func(node *html.Node) {
		switch node.Data {
		case "input":
			name := attribute(node, "name")
			if name != "" {
				form.values.Set(name, attribute(node, "value"))
			}
		case "img":
			if hasClass(node, "captcha") {
				form.captchaPath = attribute(node, "src")
			}
		}
	})

	for _, name := range []string{"next", "csrfmiddlewaretoken", "captcha_0"} {
		if form.values.Get(name) == "" {
			return casLoginForm{}, fmt.Errorf("login field %q not found", name)
		}
	}
	if form.captchaPath == "" {
		return casLoginForm{}, fmt.Errorf("captcha image not found")
	}
	return form, nil
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

func resolveCASURL(base *url.URL, reference, expectedHost string) (*url.URL, error) {
	resolved, err := base.Parse(reference)
	if err != nil {
		return nil, err
	}
	if resolved.Host != expectedHost {
		return nil, fmt.Errorf("unexpected CAS host %q", resolved.Host)
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
	req.Header.Set("User-Agent", "Cairn/1.0")
	response, err := client.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return nil, response.Request.URL, fmt.Errorf("request returned HTTP %d", response.StatusCode)
	}
	body, err := util.ReadResponseBody(response, maxBytes)
	return body, response.Request.URL, err
}

// solveCaptcha 使用图鉴 API 识别验证码图像，并返回识别结果。
func (l *CourseRegistrationLogic) solveCaptcha(
	ctx context.Context,
	username string,
	password string,
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
		TypeID:   11,
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

func postCASLogin(
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
	req.Header.Set("Referer", referer.String())
	req.Header.Set("User-Agent", "Cairn/1.0")
	return client.Do(req)
}

func samePage(actual *url.URL, expected string) bool {
	expectedURL, err := url.Parse(expected)
	if err != nil {
		return false
	}
	return actual.Scheme == expectedURL.Scheme && actual.Host == expectedURL.Host && actual.Path == expectedURL.Path
}
