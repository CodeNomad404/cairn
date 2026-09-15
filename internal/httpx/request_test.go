package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

type validReq struct {
	Slug string `json:"slug"`
}

func (v validReq) Validate() error { return nil }

type invalidReq struct{}

func (i invalidReq) Validate() error { return errors.New("bad") }

func parseBodyRequest(t *testing.T, body string, maxBytes int64) (*validReq, *httptest.ResponseRecorder, error) {
	t.Helper()
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req, err := ParseBody[validReq](w, r, maxBytes)
	return req, w, err
}

func wantAppError(t *testing.T, err error, message string, wantCause bool) {
	t.Helper()
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("err = %v, want *AppError", err)
	}
	if appErr.Message != message {
		t.Fatalf("message = %q, want %q", appErr.Message, message)
	}
	if (appErr.Unwrap() != nil) != wantCause {
		t.Fatalf("cause present = %v, want %v (err = %v)", appErr.Unwrap() != nil, wantCause, err)
	}
}

func TestParseBody_Success(t *testing.T) {
	req, _, err := parseBodyRequest(t, `{"slug":"hello"}`, 1<<20)
	if err != nil {
		t.Fatalf("err = %v", err)
	}
	if req.Slug != "hello" {
		t.Fatalf("slug = %q", req.Slug)
	}
}

func TestParseBody_InvalidJSON(t *testing.T) {
	req, _, err := parseBodyRequest(t, `not-json`, 1<<20)
	if req != nil {
		t.Fatalf("req = %#v, want nil", req)
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("err = %v, want *AppError", err)
	}
	if appErr.Message == "" {
		t.Fatal("message is empty")
	}
	var syntaxErr *json.SyntaxError
	if !errors.As(appErr.Unwrap(), &syntaxErr) {
		t.Fatalf("cause = %T, want *json.SyntaxError", appErr.Unwrap())
	}
}

func TestParseBody_UnknownField(t *testing.T) {
	req, _, err := parseBodyRequest(t, `{"slug":"a","unexpected":true}`, 1<<20)
	if req != nil {
		t.Fatalf("req = %#v, want nil", req)
	}
	var appErr *AppError
	if !errors.As(err, &appErr) {
		t.Fatalf("err = %v, want *AppError", err)
	}
	if !strings.Contains(appErr.Message, "unknown field") {
		t.Fatalf("message = %q, want it to mention the unknown field", appErr.Message)
	}
	if appErr.Unwrap() == nil {
		t.Fatal("cause is nil")
	}
}

func TestParseBody_TrailingGarbage(t *testing.T) {
	req, _, err := parseBodyRequest(t, `{"slug":"a"} {"extra":"value"}`, 1<<20)
	if req != nil {
		t.Fatalf("req = %#v, want nil", req)
	}
	wantAppError(t, err, "request body must contain a single JSON value", true)
}

func TestParseBody_BodyTooLarge(t *testing.T) {
	req, _, err := parseBodyRequest(t, `{"slug":"`+strings.Repeat("x", 100)+`"}`, 4)
	if req != nil {
		t.Fatalf("req = %#v, want nil", req)
	}
	wantAppError(t, err, "http: request body too large", true)
	var maxErr *http.MaxBytesError
	if !errors.As(err, &maxErr) {
		t.Fatalf("cause = %T, want *http.MaxBytesError", err)
	}
}

func TestParseBody_ValidationFailed(t *testing.T) {
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(`{}`))
	req, err := ParseBody[invalidReq](w, r, 1<<20)
	if req != nil {
		t.Fatalf("req = %#v, want nil", req)
	}
	wantAppError(t, err, "bad", true)
}

func withRouteParam(t *testing.T, name, value string) *http.Request {
	t.Helper()
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add(name, value)
	r := httptest.NewRequest(http.MethodGet, "/", nil)
	return r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))
}

func TestParsePath(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    string
		wantErr bool
	}{
		{"success", "hello", "hello", false},
		{"trimmed", "  hello  ", "hello", false},
		{"missing", "", "", true},
		{"empty", "  ", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := withRouteParam(t, "slug", tt.value)
			got, err := ParsePath(r, "slug")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("value = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestParsePath_MissingErrorContract(t *testing.T) {
	r := withRouteParam(t, "slug", "")
	_, err := ParsePath(r, "slug")
	wantAppError(t, err, "参数错误", false)
}

func TestParsePositiveInt64Path(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		want    int64
		wantErr bool
	}{
		{"success", "42", 42, false},
		{"non-numeric", "abc", 0, true},
		{"zero", "0", 0, true},
		{"negative", "-1", 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := withRouteParam(t, "id", tt.value)
			got, err := ParsePositiveInt64Path(r, "id")
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("value = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestParsePositiveInt64Path_ConversionErrorInCause(t *testing.T) {
	r := withRouteParam(t, "id", "abc")
	_, err := ParsePositiveInt64Path(r, "id")
	wantAppError(t, err, "参数错误", true)
	var numErr *strconv.NumError
	if !errors.As(err, &numErr) {
		t.Fatalf("cause = %T, want *strconv.NumError", err)
	}
}

func TestParsePagination(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		want    Pagination
		wantErr bool
	}{
		{"defaults", "", Pagination{Page: 1, PageSize: 20}, false},
		{"valid", "?page=2&page_size=30", Pagination{Page: 2, PageSize: 30}, false},
		{"lower_bounds", "?page=1&page_size=1", Pagination{Page: 1, PageSize: 1}, false},
		{"upper_bound", "?page_size=100", Pagination{Page: 1, PageSize: 100}, false},
		{"invalid_page_zero", "?page=0", Pagination{}, true},
		{"invalid_page_negative", "?page=-1", Pagination{}, true},
		{"invalid_page_non_numeric", "?page=abc", Pagination{}, true},
		{"invalid_page_size_zero", "?page_size=0", Pagination{}, true},
		{"invalid_page_size_non_numeric", "?page_size=abc", Pagination{}, true},
		{"page_size_over_limit", "?page_size=101", Pagination{}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/posts"+tt.query, nil)
			got, err := ParsePagination(r)
			if (err != nil) != tt.wantErr {
				t.Fatalf("err = %v, wantErr %v", err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("pagination = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestParsePagination_ErrorContract(t *testing.T) {
	r := httptest.NewRequest(http.MethodGet, "/posts?page=abc", nil)
	_, err := ParsePagination(r)
	wantAppError(t, err, "参数错误", true)
}

func TestParseQueryValues(t *testing.T) {
	location := time.FixedZone("test", 8*60*60)
	defaultDate := time.Date(2026, time.September, 15, 10, 0, 0, 0, location)
	r := httptest.NewRequest(
		http.MethodGet,
		"/?period=3&building_id=%20SY%20&date=2026-09-16",
		nil,
	)

	period, err := ParseQueryInt(r, "period", 0, 1, 7)
	if err != nil || period != 3 {
		t.Fatalf("period = %d, err = %v", period, err)
	}
	buildingID, err := ParseQueryString(r, "building_id", 16)
	if err != nil || buildingID != "SY" {
		t.Fatalf("building ID = %q, err = %v", buildingID, err)
	}
	date, err := ParseQueryDate(r, "date", defaultDate)
	if err != nil || date.Format(time.DateOnly) != "2026-09-16" || date.Location() != location {
		t.Fatalf("date = %v, err = %v", date, err)
	}
}

func TestParseQueryValuesRejectInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		query string
		parse func(*http.Request) error
	}{
		{
			name: "integer range", query: "?period=8",
			parse: func(r *http.Request) error {
				_, err := ParseQueryInt(r, "period", 0, 1, 7)
				return err
			},
		},
		{
			name: "string length", query: "?building_id=too-long",
			parse: func(r *http.Request) error {
				_, err := ParseQueryString(r, "building_id", 3)
				return err
			},
		},
		{
			name: "date format", query: "?date=2026/09/15",
			parse: func(r *http.Request) error {
				_, err := ParseQueryDate(r, "date", time.Now())
				return err
			},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/"+test.query, nil)
			if err := test.parse(r); err == nil {
				t.Fatal("invalid query accepted")
			}
		})
	}
}
