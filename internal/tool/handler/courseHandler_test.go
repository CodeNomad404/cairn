package handler

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/tool/logic"
)

func TestCourseRegistrationLoginRoute_RequiresCairnUser(t *testing.T) {
	router := chi.NewRouter()
	handler := NewCourseHandler(
		logic.NewCourseRegistrationLogic(nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil))),
	)
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(
		http.MethodPost,
		"/tools/course-registration/login",
		strings.NewReader(`not-json`),
	)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestCourseRegistrationLoginRequestValidate(t *testing.T) {
	valid := CourseRegistrationLoginRequest{
		StudentID:      "24280000",
		Password:       "school-password",
		TujianUsername: "tujian-user",
		TujianPassword: "tujian-password",
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}
	if err := (CourseRegistrationLoginRequest{}).Validate(); err != nil {
		t.Fatalf("empty request for saved credentials rejected: %v", err)
	}

	tests := []struct {
		name   string
		mutate func(*CourseRegistrationLoginRequest)
	}{
		{name: "student ID", mutate: func(r *CourseRegistrationLoginRequest) { r.StudentID = " " }},
		{name: "school password", mutate: func(r *CourseRegistrationLoginRequest) { r.Password = "" }},
		{name: "Tujian username", mutate: func(r *CourseRegistrationLoginRequest) { r.TujianUsername = " " }},
		{name: "Tujian password", mutate: func(r *CourseRegistrationLoginRequest) { r.TujianPassword = "" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := valid
			test.mutate(&request)
			if err := request.Validate(); err == nil {
				t.Fatal("Validate error = nil")
			}
		})
	}
}

func TestCourseGrabRequestValidate(t *testing.T) {
	valid := CourseGrabRequest{
		Courses: []string{"M302006B:计算机体系结构 01 计算机学院"},
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("valid request rejected: %v", err)
	}

	invalid := []CourseGrabRequest{
		{},
		{Courses: []string{" "}},
	}
	for _, request := range invalid {
		if err := request.Validate(); err == nil {
			t.Fatalf("invalid request accepted: %#v", request)
		}
	}
}
