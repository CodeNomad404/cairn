package handler

import (
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/tool/logic"
)

func TestClassroomListRouteRequiresCairnUserBeforeParsingQuery(t *testing.T) {
	router := chi.NewRouter()
	handler := NewClassroomHandler(logic.NewClassroomLogic(
		nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)),
	))
	handler.RegisterRoutes(router)

	request := httptest.NewRequest(http.MethodGet, "/tool/classroom/list?period=invalid", nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)

	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}
