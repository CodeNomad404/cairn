package logic

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"cairn.cn/internal/httpx"
	"cairn.cn/internal/tool"
	toolmodel "cairn.cn/internal/tool/model"
)

type fakeClassroomAvailabilityRepo struct {
	items []toolmodel.ClassroomAvailability
}

func (f *fakeClassroomAvailabilityRepo) ReplaceSnapshot(
	_ context.Context,
	items []toolmodel.ClassroomAvailability,
) error {
	for _, item := range items {
		updated := false
		for index, existing := range f.items {
			if isoWeekday(existing.Date) == isoWeekday(item.Date) &&
				existing.Period == item.Period && existing.BuildingID == item.BuildingID &&
				existing.RoomName == item.RoomName {
				f.items[index] = item
				updated = true
				break
			}
		}
		if !updated {
			f.items = append(f.items, item)
		}
	}
	return nil
}

func (f *fakeClassroomAvailabilityRepo) List(
	_ context.Context,
	query tool.ClassroomAvailabilityQuery,
) ([]toolmodel.ClassroomAvailability, *time.Time, error) {
	items := make([]toolmodel.ClassroomAvailability, 0)
	var syncedAt *time.Time
	for _, item := range f.items {
		if isoWeekday(item.Date) == query.Weekday {
			value := item.SyncedAt
			if syncedAt == nil || value.After(*syncedAt) {
				syncedAt = &value
			}
		}
		if isoWeekday(item.Date) == query.Weekday && (query.Period == 0 || item.Period == query.Period) &&
			item.Capacity >= query.MinCapacity &&
			(query.BuildingID == "" || item.BuildingID == query.BuildingID) {
			items = append(items, item)
		}
	}
	return items, syncedAt, nil
}

func (f *fakeClassroomAvailabilityRepo) LastSyncedAt(context.Context) (*time.Time, error) {
	if len(f.items) == 0 {
		return nil, nil
	}
	value := f.items[0].SyncedAt
	return &value, nil
}

func TestParseClassroomBuildings(t *testing.T) {
	html := `<select name="jxlh">
		<option value="">-- 教学楼 --</option>
		<option value="1">思源楼</option>
		<option value="11">逸夫教学楼</option>
	</select>`
	buildings, err := parseClassroomBuildings(strings.NewReader(html))
	if err != nil {
		t.Fatalf("parse buildings: %v", err)
	}
	if len(buildings) != 2 || buildings[0].ID != "1" || buildings[1].Name != "逸夫教学楼" {
		t.Fatalf("buildings = %#v", buildings)
	}
}

func TestParseClassroomAvailability(t *testing.T) {
	var page strings.Builder
	page.WriteString(`<table class="table table-bordered"><tr><th>星期</th>`)
	for day := 14; day <= 20; day++ {
		fmt.Fprintf(&page, `<th>星期 <span>09月%d日</span></th>`, day)
	}
	page.WriteString(`</tr><tr><th>教室/节次</th>`)
	for range 49 {
		page.WriteString(`<td>1</td>`)
	}
	page.WriteString(`</tr><tr><td>SY101 <span>(90)</span></td>`)
	colors := []string{"#fff", "#e46868", "#9e6868", "#394ed6", "#77bf6d", "#d8cc56", "#ffffff"}
	for index := range 49 {
		fmt.Fprintf(&page, `<td style="background-color: %s"></td>`, colors[index%len(colors)])
	}
	page.WriteString(`</tr></table>`)

	syncedAt := time.Date(2026, time.September, 15, 1, 0, 0, 0, classroomLocation)
	items, err := parseClassroomAvailability(
		strings.NewReader(page.String()),
		classroomBuilding{ID: "1", Name: "思源楼"},
		syncedAt,
	)
	if err != nil {
		t.Fatalf("parse availability: %v", err)
	}
	if len(items) != 49 {
		t.Fatalf("item count = %d, want 49", len(items))
	}
	if got := items[0]; got.Date.Format(time.DateOnly) != "2026-09-14" || got.Period != 1 ||
		got.RoomName != "SY101" || got.Capacity != 90 || got.OccupationType != "free" {
		t.Fatalf("first item = %#v", got)
	}
	if items[7].Date.Format(time.DateOnly) != "2026-09-15" || items[7].Period != 1 {
		t.Fatalf("eighth item = %#v", items[7])
	}
	if items[1].OccupationType != "scheduled" || items[5].OccupationType != "other" {
		t.Fatalf("occupation mapping = %q, %q", items[1].OccupationType, items[5].OccupationType)
	}
}

func TestParseClassroomDateAcrossYearBoundary(t *testing.T) {
	reference := time.Date(2026, time.December, 31, 23, 0, 0, 0, classroomLocation)
	date, err := parseClassroomDate("星期五 01月01日", reference)
	if err != nil {
		t.Fatalf("parse date: %v", err)
	}
	if got := date.Format(time.DateOnly); got != "2027-01-01" {
		t.Fatalf("date = %s, want 2027-01-01", got)
	}
}

func TestClassroomOccupationTypeRejectsUnknownColor(t *testing.T) {
	if _, err := classroomOccupationType("background-color: #123456"); err == nil {
		t.Fatal("unknown color accepted")
	}
}

func TestParseClassroomRoomAllowsUnknownCapacity(t *testing.T) {
	name, capacity, err := parseClassroomRoom("YF东707 (-)")
	if err != nil || name != "YF东707" || capacity != 0 {
		t.Fatalf("name = %q, capacity = %d, err = %v", name, capacity, err)
	}
}

func TestParseClassroomAvailabilityAllowsEmptyBuilding(t *testing.T) {
	page := classroomTestPage(time.Now().In(classroomLocation))
	roomStart := strings.Index(page, "<tr><td>SY101")
	if roomStart < 0 {
		t.Fatal("test room row not found")
	}
	page = page[:roomStart] + "</table>"
	items, err := parseClassroomAvailability(
		strings.NewReader(page), classroomBuilding{ID: "100", Name: "学生活动服务中心"}, time.Now(),
	)
	if err != nil {
		t.Fatalf("empty building rejected: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("items = %d, want 0", len(items))
	}
}

func TestGetClassroomPageRetriesTemporaryStatus(t *testing.T) {
	var requests atomic.Int32
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		if requests.Add(1) < 3 {
			http.Error(w, "temporary", http.StatusServiceUnavailable)
			return
		}
		_, _ = w.Write([]byte("ok"))
	}))
	defer server.Close()

	body, _, err := getClassroomPage(context.Background(), server.Client(), server.URL)
	if err != nil {
		t.Fatalf("temporary status was not retried: %v", err)
	}
	if string(body) != "ok" || requests.Load() != 3 {
		t.Fatalf("body = %q, requests = %d", body, requests.Load())
	}
}

func TestNextClassroomSyncAt(t *testing.T) {
	tests := []struct {
		now  string
		want string
	}{
		{now: "2026-09-15T00:30:00+08:00", want: "2026-09-15T01:00:00+08:00"},
		{now: "2026-09-15T01:00:00+08:00", want: "2026-09-16T01:00:00+08:00"},
		{now: "2026-09-15T10:00:00+08:00", want: "2026-09-16T01:00:00+08:00"},
	}
	for _, test := range tests {
		now, err := time.Parse(time.RFC3339, test.now)
		if err != nil {
			t.Fatal(err)
		}
		if got := nextClassroomSyncAt(now).Format(time.RFC3339); got != test.want {
			t.Fatalf("next sync = %s, want %s", got, test.want)
		}
	}
}

func TestClassroomListRefreshesStaleWeekdayAndPreservesOtherWeekdays(t *testing.T) {
	today := time.Now().In(classroomLocation)
	lastWeek := today.AddDate(0, 0, -7)
	yesterday := today.AddDate(0, 0, -1)
	_, encodedPublicKey := testSSOKey(t)
	escapedPublicKey := strings.ReplaceAll(encodedPublicKey, "/", `\/`)
	var buildingRequests atomic.Int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/home/":
			if _, err := r.Cookie("session"); err == nil {
				_, _ = w.Write([]byte("home"))
				return
			}
			http.Redirect(w, r, server.URL+"/o/authorize/", http.StatusFound)
		case "/o/authorize/":
			_, _ = w.Write([]byte(`<form id="loginForm" method="post" action="/sso/login">
				<input name="loginType" value="username_password"><input name="flowId" value="flow-id">
				<input name="client_id" value="mis-client"><input name="redirect_uri" value="/auth/ncallback/">
				<input name="response_type" value="code">
			</form><script>var ssoConfig = {"sm2":{"enabled":true,"publicKey":"` + escapedPublicKey + `"}};</script>`))
		case "/sso/login":
			http.SetCookie(w, &http.Cookie{Name: "session", Value: "ok", Path: "/"})
			http.Redirect(w, r, server.URL+"/auth/ncallback/?code=oauth-code", http.StatusFound)
		case "/auth/ncallback/":
			http.Redirect(w, r, server.URL+"/home/", http.StatusFound)
		case "/module/module/10/":
			_, _ = w.Write([]byte(`<a href="/academic-entry/">教务系统</a>`))
		case "/academic-entry/":
			_, _ = w.Write([]byte("academic home"))
		case "/classroom/timeholdresult/room_view/":
			if r.URL.Query().Get("jxlh") == "" {
				_, _ = w.Write([]byte(`<select name="jxlh"><option value="1">思源楼</option></select>`))
				return
			}
			buildingRequests.Add(1)
			_, _ = w.Write([]byte(classroomTestPage(today)))
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	credentialsRepo := &fakeCourseRegistrationCredentialsRepo{credentials: &toolmodel.CourseRegistrationCredentials{
		UserID: 7, StudentID: "student", BJTUPassword: "password",
		TujianUsername: "captcha-user", TujianPassword: "captcha-password",
	}}
	courseLogic := testCourseRegistrationLogic(t, server.URL, credentialsRepo)
	classroomRepo := &fakeClassroomAvailabilityRepo{items: []toolmodel.ClassroomAvailability{
		{
			Date: lastWeek, Period: 1, BuildingID: "1", BuildingName: "思源楼",
			RoomName: "SY101", Capacity: 90, OccupationType: "scheduled", SyncedAt: lastWeek,
		},
		{
			Date: yesterday, Period: 1, BuildingID: "2", BuildingName: "逸夫楼",
			RoomName: "YF101", Capacity: 60, OccupationType: "exam", SyncedAt: yesterday,
		},
	}}
	classroomLogic := NewClassroomLogic(
		classroomRepo, courseLogic, slog.New(slog.NewTextHandler(io.Discard, nil)),
	)
	classroomLogic.classroomURL = server.URL + "/classroom/timeholdresult/room_view/"

	result, code, err := classroomLogic.List(context.Background(), 7, today, 1, "", 0)
	if err != nil || code != httpx.CodeSuccess {
		t.Fatalf("List code = %d, err = %v", code, err)
	}
	if len(result.Rooms) != 1 || result.Rooms[0].RoomName != "SY101" || result.SyncedAt == nil {
		t.Fatalf("result = %#v", result)
	}
	if buildingRequests.Load() != 1 {
		t.Fatalf("building requests = %d, want 1", buildingRequests.Load())
	}
	if len(classroomRepo.items) != 8 {
		t.Fatalf("stored records = %d, want 7 refreshed periods and 1 other-weekday record", len(classroomRepo.items))
	}
	otherWeekdayPreserved := false
	for _, item := range classroomRepo.items {
		if item.BuildingID == "2" && item.RoomName == "YF101" {
			otherWeekdayPreserved = sameLocalDate(item.Date, yesterday) && item.OccupationType == "exam"
		}
		if item.BuildingID == "1" && item.RoomName == "SY101" && item.Period == 1 &&
			(item.OccupationType != "free" || !sameLocalDate(item.SyncedAt, today)) {
			t.Fatalf("same weekday was not refreshed: %#v", item)
		}
	}
	if !otherWeekdayPreserved {
		t.Fatal("synchronization removed or changed another weekday")
	}

	allPeriods, _, err := classroomLogic.List(context.Background(), 7, today, 0, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(allPeriods.Rooms) != 7 {
		t.Fatalf("all-period room count = %d, want 7", len(allPeriods.Rooms))
	}
	for period, room := range allPeriods.Rooms {
		if room.Period != period+1 || room.OccupationType != "free" {
			t.Fatalf("all-period room %d = %#v", period, room)
		}
	}

	_, _, err = classroomLogic.List(context.Background(), 7, today, 1, "", 0)
	if err != nil {
		t.Fatal(err)
	}
	if buildingRequests.Load() != 1 {
		t.Fatalf("cache hit triggered another synchronization")
	}
}

func classroomTestPage(start time.Time) string {
	var page strings.Builder
	page.WriteString(`<table class="table table-bordered"><tr><th>星期</th>`)
	for offset := range 7 {
		date := start.AddDate(0, 0, offset)
		fmt.Fprintf(&page, `<th>%02d月%02d日</th>`, date.Month(), date.Day())
	}
	page.WriteString(`</tr><tr><th>教室/节次</th>`)
	for range 49 {
		page.WriteString(`<td>1</td>`)
	}
	page.WriteString(`</tr><tr><td>SY101 <span>(90)</span></td>`)
	for range 49 {
		page.WriteString(`<td style="background-color: #fff"></td>`)
	}
	page.WriteString(`</tr></table>`)
	return page.String()
}
