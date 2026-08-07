package request

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cairn.cn/internal/response"
)

type validReq struct {
	Slug string `json:"slug"`
}

func (v validReq) Validate() error { return nil }

type invalidReq struct{}

func (i invalidReq) Validate() error { return errors.New("bad") }

func TestWithValidatedJSON_Success(t *testing.T) {
	called := false
	handler := WithValidatedJSON[validReq](1<<20, func(w http.ResponseWriter, r *http.Request, req *validReq) {
		called = true
		if req.Slug != "hello" {
			t.Fatalf("slug = %q", req.Slug)
		}
		w.WriteHeader(http.StatusOK)
	})

	body := strings.NewReader(`{"slug":"hello"}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if !called {
		t.Fatal("next was not called")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
}

func TestWithValidatedJSON_InvalidJSON(t *testing.T) {
	called := false
	handler := WithValidatedJSON[validReq](1<<20, func(w http.ResponseWriter, r *http.Request, req *validReq) {
		called = true
	})

	body := strings.NewReader(`not-json`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if called {
		t.Fatal("next was called for invalid JSON")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Code != int(response.CodeBadRequest) {
		t.Fatalf("code = %d", resp.Code)
	}
}

func TestWithValidatedJSON_UnknownField(t *testing.T) {
	called := false
	handler := WithValidatedJSON[validReq](1<<20, func(w http.ResponseWriter, r *http.Request, req *validReq) {
		called = true
	})

	body := strings.NewReader(`{"slug":"a","unexpected":true}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if called {
		t.Fatal("next was called for unknown field")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestWithValidatedJSON_TrailingGarbage(t *testing.T) {
	called := false
	handler := WithValidatedJSON[validReq](1<<20, func(w http.ResponseWriter, r *http.Request, req *validReq) {
		called = true
	})

	body := strings.NewReader(`{"slug":"a"} {"extra":"value"}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if called {
		t.Fatal("next was called for trailing garbage")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestWithValidatedJSON_BodyTooLarge(t *testing.T) {
	called := false
	handler := WithValidatedJSON[validReq](4, func(w http.ResponseWriter, r *http.Request, req *validReq) {
		called = true
	})

	body := strings.NewReader(`{"slug":"` + strings.Repeat("x", 100) + `"}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if called {
		t.Fatal("next was called for oversized body")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestWithValidatedJSON_ValidationFailed(t *testing.T) {
	called := false
	handler := WithValidatedJSON[invalidReq](1<<20, func(w http.ResponseWriter, r *http.Request, req *invalidReq) {
		called = true
	})

	body := strings.NewReader(`{}`)
	r := httptest.NewRequest(http.MethodPost, "/", body)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if called {
		t.Fatal("next was called for invalid request")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
