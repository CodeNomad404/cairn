package logic

import (
	"context"
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

	"golang.org/x/net/html"

	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
)

const defaultAAClassroomURL = "https://aa.bjtu.edu.cn/classroom/timeholdresult/room_view/"

const classroomRequestInterval = 50 * time.Millisecond

const classroomSyncHour = 3

var (
	classroomDatePattern  = regexp.MustCompile(`(\d{1,2})月(\d{1,2})日`)
	classroomRoomPattern  = regexp.MustCompile(`^(.+?)\s*\((\d+|-)\)$`)
	classroomColorPattern = regexp.MustCompile(`(?i)background-color\s*:\s*(#[0-9a-f]{3,6})`)
	classroomLocation     = time.FixedZone("Asia/Shanghai", 8*60*60)
)

type classroomBuilding struct {
	ID   string
	Name string
}

type ClassroomRoomResult struct {
	Period         int    `json:"period"`
	BuildingID     string `json:"building_id"`
	BuildingName   string `json:"building_name"`
	RoomName       string `json:"room_name"`
	Capacity       int    `json:"capacity"`
	OccupationType string `json:"occupation_type"`
}

type ClassroomListResult struct {
	Date     string                `json:"date"`
	Period   int                   `json:"period,omitempty"`
	SyncedAt *time.Time            `json:"synced_at,omitempty"`
	Rooms    []ClassroomRoomResult `json:"rooms"`
}

type ClassroomLogic struct {
	repo         tool.ClassroomAvailabilityRepo
	courseLogic  *CourseRegistrationLogic
	logger       *slog.Logger
	classroomURL string
	syncMu       sync.Mutex
}

func NewClassroomLogic(
	repo tool.ClassroomAvailabilityRepo,
	courseLogic *CourseRegistrationLogic,
	logger *slog.Logger,
) *ClassroomLogic {
	return &ClassroomLogic{
		repo: repo, courseLogic: courseLogic, logger: logger,
		classroomURL: defaultAAClassroomURL,
	}
}

func (l *ClassroomLogic) List(
	ctx context.Context,
	userID int64,
	date time.Time,
	period int,
	buildingID string,
	minCapacity int,
) (ClassroomListResult, httpx.Code, error) {
	query := tool.ClassroomAvailabilityQuery{
		Weekday: isoWeekday(date), Period: period, BuildingID: buildingID, MinCapacity: minCapacity,
	}
	items, syncedAt, err := l.repo.List(ctx, query)
	if err != nil {
		return ClassroomListResult{}, httpx.CodeInternalError,
			httpx.NewAppError("读取教室列表失败", err)
	}
	now := time.Now().In(classroomLocation)
	if sameLocalDate(date, now) && (syncedAt == nil || !sameLocalDate(*syncedAt, now)) {
		code, err := l.ensureWeekdaySynced(ctx, userID, query)
		if err != nil {
			return ClassroomListResult{}, code, err
		}
		items, syncedAt, err = l.repo.List(ctx, query)
		if err != nil {
			return ClassroomListResult{}, httpx.CodeInternalError,
				httpx.NewAppError("读取教室列表失败", err)
		}
	}
	rooms := make([]ClassroomRoomResult, 0, len(items))
	for _, item := range items {
		rooms = append(rooms, ClassroomRoomResult{
			Period:     item.Period,
			BuildingID: item.BuildingID, BuildingName: item.BuildingName,
			RoomName: item.RoomName, Capacity: item.Capacity,
			OccupationType: item.OccupationType,
		})
	}
	return ClassroomListResult{
		Date: date.Format(time.DateOnly), Period: period, SyncedAt: syncedAt, Rooms: rooms,
	}, httpx.CodeSuccess, nil
}

// StartScheduler 启动教室同步调度器。
func (l *ClassroomLogic) StartScheduler(ctx context.Context, userID int64) {
	// 如果 userID 小于等于 0，则不启动调度器。
	if userID <= 0 {
		return
	}
	go l.runScheduler(ctx, userID)
}

func (l *ClassroomLogic) runScheduler(ctx context.Context, userID int64) {
	// 在应用启动时，如果当前时间在 3 点之后且当天尚未同步，则立即执行一次同步。
	now := time.Now().In(classroomLocation)
	lastSyncedAt, err := l.repo.LastSyncedAt(ctx)
	if err != nil {
		l.logger.ErrorContext(ctx, "read classroom synchronization status", "error", err)
	} else if now.Hour() >= classroomSyncHour && (lastSyncedAt == nil || !sameLocalDate(*lastSyncedAt, now)) {
		l.synchronizeAndLog(ctx, userID)
	}

	// 每天 3 点执行一次同步。
	for {
		now = time.Now().In(classroomLocation)
		next := nextClassroomSyncAt(now)
		timer := time.NewTimer(time.Until(next))
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
			l.synchronizeAndLog(ctx, userID)
		}
	}
}

func nextClassroomSyncAt(now time.Time) time.Time {
	now = now.In(classroomLocation)
	next := time.Date(now.Year(), now.Month(), now.Day(), classroomSyncHour, 0, 0, 0, classroomLocation)
	if !next.After(now) {
		next = next.AddDate(0, 0, 1)
	}
	return next
}

func (l *ClassroomLogic) synchronizeAndLog(ctx context.Context, userID int64) {
	count, _, err := l.Synchronize(ctx, userID)
	if err != nil {
		l.logger.ErrorContext(ctx, "classroom synchronization failed", "error", err)
		return
	}
	l.logger.InfoContext(ctx, "classroom synchronization completed", "records", count)
}

func (l *ClassroomLogic) ensureWeekdaySynced(
	ctx context.Context,
	userID int64,
	query tool.ClassroomAvailabilityQuery,
) (httpx.Code, error) {
	l.syncMu.Lock()
	defer l.syncMu.Unlock()

	_, syncedAt, err := l.repo.List(ctx, query)
	if err != nil {
		return httpx.CodeInternalError, httpx.NewAppError("读取教室列表失败", err)
	}
	if syncedAt != nil && sameLocalDate(*syncedAt, time.Now().In(classroomLocation)) {
		return httpx.CodeSuccess, nil
	}
	_, code, err := l.synchronize(ctx, userID)
	return code, err
}

func (l *ClassroomLogic) Synchronize(
	ctx context.Context,
	userID int64,
) (int, httpx.Code, error) {
	l.syncMu.Lock()
	defer l.syncMu.Unlock()
	return l.synchronize(ctx, userID)
}

func (l *ClassroomLogic) synchronize(
	ctx context.Context,
	userID int64,
) (int, httpx.Code, error) {
	if _, code, err := l.courseLogic.Login(ctx, userID, "", "", "", ""); err != nil {
		return 0, code, fmt.Errorf("login to BJTU: %w", err)
	}
	client, code, err := l.courseLogic.openAcademicSystem(ctx, userID)
	if err != nil {
		return 0, code, fmt.Errorf("open academic system: %w", err)
	}

	body, currentURL, err := getPage(ctx, client, l.classroomURL, maxHTMLBytes)
	if err != nil {
		return 0, httpx.CodeInternalError, fmt.Errorf("open classroom page: %w", err)
	}
	if currentURL.Host != l.courseLogic.aaHost || !strings.HasPrefix(currentURL.Path, "/classroom/") {
		return 0, httpx.CodeInternalError, fmt.Errorf("unexpected classroom page %q", currentURL.Redacted())
	}
	buildings, err := parseClassroomBuildings(strings.NewReader(string(body)))
	if err != nil {
		return 0, httpx.CodeInternalError, err
	}

	syncedAt := time.Now().In(classroomLocation)
	items := make([]toolmodel.ClassroomAvailability, 0, 20000)
	for _, building := range buildings {
		buildingURL := cloneURL(currentURL)
		query := buildingURL.Query()
		query.Set("jxlh", building.ID)
		query.Set("page", "1")
		query.Set("perpage", "500")
		buildingURL.RawQuery = query.Encode()

		buildingBody, finalURL, err := getClassroomPage(ctx, client, buildingURL.String())
		if err != nil {
			return 0, httpx.CodeInternalError, fmt.Errorf("fetch building %s: %w", building.ID, err)
		}
		if finalURL.Host != l.courseLogic.aaHost || !strings.HasPrefix(finalURL.Path, "/classroom/") {
			return 0, httpx.CodeUnauthorized, httpx.NewAppError("BJTU academic system session expired", nil)
		}
		buildingItems, err := parseClassroomAvailability(
			strings.NewReader(string(buildingBody)), building, syncedAt,
		)
		if err != nil {
			return 0, httpx.CodeInternalError, fmt.Errorf("parse building %s: %w", building.ID, err)
		}
		items = append(items, classroomAvailabilityForDate(buildingItems, syncedAt)...)
		if err := waitForClassroomRequest(ctx, classroomRequestInterval); err != nil {
			return 0, httpx.CodeInternalError, err
		}
	}
	if err := l.repo.ReplaceSnapshot(ctx, items); err != nil {
		return 0, httpx.CodeInternalError, fmt.Errorf("save classroom snapshot: %w", err)
	}
	return len(items), httpx.CodeSuccess, nil
}

func classroomAvailabilityForDate(
	items []toolmodel.ClassroomAvailability,
	date time.Time,
) []toolmodel.ClassroomAvailability {
	filtered := make([]toolmodel.ClassroomAvailability, 0, len(items)/7)
	for _, item := range items {
		if sameLocalDate(item.Date, date) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func getClassroomPage(
	ctx context.Context,
	client *http.Client,
	endpoint string,
) ([]byte, *url.URL, error) {
	var finalURL *url.URL
	var err error
	for attempt := range 3 {
		var body []byte
		body, finalURL, err = getPage(ctx, client, endpoint, maxHTMLBytes)
		if err == nil {
			return body, finalURL, nil
		}
		var statusErr *httpStatusError
		if !errors.As(err, &statusErr) || statusErr.StatusCode != http.StatusTooManyRequests && statusErr.StatusCode < 500 {
			return nil, finalURL, err
		}
		if attempt < 2 {
			if waitErr := waitForClassroomRequest(ctx, time.Duration(200*(1<<attempt))*time.Millisecond); waitErr != nil {
				return nil, finalURL, waitErr
			}
		}
	}
	return nil, finalURL, err
}

func waitForClassroomRequest(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func parseClassroomBuildings(r io.Reader) ([]classroomBuilding, error) {
	document, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	selectNode := findFirstElement(document, func(node *html.Node) bool {
		return node.Data == "select" && attribute(node, "name") == "jxlh"
	})
	if selectNode == nil {
		return nil, fmt.Errorf("building selector not found")
	}
	buildings := make([]classroomBuilding, 0)
	walkElements(selectNode, func(node *html.Node) {
		if node.Data != "option" {
			return
		}
		id := strings.TrimSpace(attribute(node, "value"))
		name := strings.TrimSpace(elementText(node))
		if id != "" && name != "" {
			buildings = append(buildings, classroomBuilding{ID: id, Name: name})
		}
	})
	if len(buildings) == 0 {
		return nil, fmt.Errorf("no buildings found")
	}
	return buildings, nil
}

func parseClassroomAvailability(
	r io.Reader,
	building classroomBuilding,
	syncedAt time.Time,
) ([]toolmodel.ClassroomAvailability, error) {
	document, err := html.Parse(r)
	if err != nil {
		return nil, err
	}
	table := findFirstElement(document, func(node *html.Node) bool {
		return node.Data == "table" && hasClass(node, "table-bordered")
	})
	if table == nil {
		return nil, fmt.Errorf("availability table not found")
	}
	rows := collectElements(table, "tr")
	if len(rows) < 2 {
		return nil, fmt.Errorf("availability table headers not found")
	}

	headerCells := directCells(rows[0])
	if len(headerCells) != 8 {
		return nil, fmt.Errorf("expected 7 date headers, got %d", len(headerCells)-1)
	}
	dates := make([]time.Time, 0, 7)
	for _, cell := range headerCells[1:] {
		date, err := parseClassroomDate(elementText(cell), syncedAt)
		if err != nil {
			return nil, err
		}
		dates = append(dates, date)
	}

	items := make([]toolmodel.ClassroomAvailability, 0, (len(rows)-2)*49)
	for _, row := range rows[2:] {
		cells := directCells(row)
		if len(cells) == 0 {
			continue
		}
		if len(cells) != 50 {
			return nil, fmt.Errorf("room row has %d cells, want 50", len(cells))
		}
		roomName, capacity, err := parseClassroomRoom(elementText(cells[0]))
		if err != nil {
			return nil, err
		}
		for index, cell := range cells[1:] {
			occupationType, err := classroomOccupationType(attribute(cell, "style"))
			if err != nil {
				return nil, fmt.Errorf("room %s cell %d: %w", roomName, index+1, err)
			}
			items = append(items, toolmodel.ClassroomAvailability{
				Date: dates[index/7], Period: index%7 + 1,
				BuildingID: building.ID, BuildingName: building.Name,
				RoomName: roomName, Capacity: capacity,
				OccupationType: occupationType, SyncedAt: syncedAt,
			})
		}
	}
	return items, nil
}

func parseClassroomDate(value string, reference time.Time) (time.Time, error) {
	match := classroomDatePattern.FindStringSubmatch(value)
	if len(match) != 3 {
		return time.Time{}, fmt.Errorf("classroom date not found in %q", strings.TrimSpace(value))
	}
	month, _ := strconv.Atoi(match[1])
	day, _ := strconv.Atoi(match[2])
	var closest time.Time
	var closestDistance time.Duration
	for year := reference.Year() - 1; year <= reference.Year()+1; year++ {
		candidate := time.Date(year, time.Month(month), day, 0, 0, 0, 0, classroomLocation)
		if candidate.Month() != time.Month(month) || candidate.Day() != day {
			continue
		}
		distance := reference.Sub(candidate)
		if distance < 0 {
			distance = -distance
		}
		if closest.IsZero() || distance < closestDistance {
			closest, closestDistance = candidate, distance
		}
	}
	if closest.IsZero() {
		return time.Time{}, fmt.Errorf("invalid classroom date %q", strings.TrimSpace(value))
	}
	return closest, nil
}

func parseClassroomRoom(value string) (string, int, error) {
	match := classroomRoomPattern.FindStringSubmatch(strings.TrimSpace(value))
	if len(match) != 3 {
		return "", 0, fmt.Errorf("invalid classroom label %q", strings.TrimSpace(value))
	}
	if match[2] == "-" {
		return strings.TrimSpace(match[1]), 0, nil
	}
	capacity, err := strconv.Atoi(match[2])
	if err != nil {
		return "", 0, err
	}
	return strings.TrimSpace(match[1]), capacity, nil
}

func classroomOccupationType(style string) (string, error) {
	match := classroomColorPattern.FindStringSubmatch(style)
	if len(match) != 2 {
		return "", fmt.Errorf("background color not found")
	}
	switch strings.ToLower(match[1]) {
	case "#fff", "#ffffff":
		return "free", nil
	case "#e46868":
		return "scheduled", nil
	case "#9e6868":
		return "rescheduled", nil
	case "#394ed6":
		return "exam", nil
	case "#77bf6d":
		return "laboratory", nil
	case "#d8cc56":
		return "other", nil
	default:
		return "", fmt.Errorf("unknown background color %q", match[1])
	}
}

func findFirstElement(node *html.Node, match func(*html.Node) bool) *html.Node {
	if node.Type == html.ElementNode && match(node) {
		return node
	}
	for child := node.FirstChild; child != nil; child = child.NextSibling {
		if found := findFirstElement(child, match); found != nil {
			return found
		}
	}
	return nil
}

func collectElements(node *html.Node, name string) []*html.Node {
	items := make([]*html.Node, 0)
	walkElements(node, func(candidate *html.Node) {
		if candidate.Data == name {
			items = append(items, candidate)
		}
	})
	return items
}

func directCells(row *html.Node) []*html.Node {
	cells := make([]*html.Node, 0)
	for child := row.FirstChild; child != nil; child = child.NextSibling {
		if child.Type == html.ElementNode && (child.Data == "th" || child.Data == "td") {
			cells = append(cells, child)
		}
	}
	return cells
}

func cloneURL(source *url.URL) *url.URL {
	cloned := *source
	return &cloned
}

func sameLocalDate(left, right time.Time) bool {
	left = left.In(classroomLocation)
	right = right.In(classroomLocation)
	return left.Year() == right.Year() && left.YearDay() == right.YearDay()
}

func isoWeekday(value time.Time) int {
	weekday := int(value.In(classroomLocation).Weekday())
	if weekday == 0 {
		return 7
	}
	return weekday
}
