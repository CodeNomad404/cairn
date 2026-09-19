package handler

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"

	"cairn.cn/internal/auth"
	authmid "cairn.cn/internal/auth/middleware"
)

func accountingRouter(handler *AccountingHandler) http.Handler {
	r := chi.NewRouter()
	handler.RegisterRoutes(r)
	return r
}

func TestAccountingRequiresAuthenticationBeforeParsingBody(t *testing.T) {
	router := accountingRouter(NewAccountingHandler(nil))
	request := httptest.NewRequest(
		http.MethodPost,
		"/tool/accounting-expense/create",
		strings.NewReader(`{"invalid"`),
	)
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

func TestCreateAccountingExpenseRejectsInvalidCategory(t *testing.T) {
	router := accountingRouter(NewAccountingHandler(nil))
	request := httptest.NewRequest(
		http.MethodPost,
		"/tool/accounting-expense/create",
		strings.NewReader(`{"book_id":1,"amount_cents":100,"category":9,"spent_at":"2026-09-19T12:00:00+08:00"}`),
	)
	request = request.WithContext(authmid.WithPrincipal(request.Context(), &auth.Principal{ID: 1}))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}

func TestMonthlyStatisticRejectsInvalidMonth(t *testing.T) {
	router := accountingRouter(NewAccountingHandler(nil))
	request := httptest.NewRequest(http.MethodGet, "/tool/accounting-statistic/get?month=2026-13", nil)
	request = request.WithContext(authmid.WithPrincipal(request.Context(), &auth.Principal{ID: 1}))
	response := httptest.NewRecorder()

	router.ServeHTTP(response, request)
	if response.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusBadRequest)
	}
}
