package util

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-chi/chi/v5"
)

type validReq struct {
	Slug string `json:"slug"`
}

func (v validReq) Validate() error { return nil }

type invalidReq struct{}

func (i invalidReq) Validate() error { return errors.New("bad") }

func TestParseBody_Success(t *testing.T) {
	called := false
	handler := ParseBody[validReq](1<<20, func(w http.ResponseWriter, r *http.Request, req *validReq) {
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

func TestParseBody_InvalidJSON(t *testing.T) {
	called := false
	handler := ParseBody[validReq](1<<20, func(w http.ResponseWriter, r *http.Request, req *validReq) {
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
	if resp.Code != int(CodeBadRequest) {
		t.Fatalf("code = %d", resp.Code)
	}
}

func TestParseBody_UnknownField(t *testing.T) {
	called := false
	handler := ParseBody[validReq](1<<20, func(w http.ResponseWriter, r *http.Request, req *validReq) {
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

func TestParseBody_TrailingGarbage(t *testing.T) {
	called := false
	handler := ParseBody[validReq](1<<20, func(w http.ResponseWriter, r *http.Request, req *validReq) {
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

func TestParseBody_BodyTooLarge(t *testing.T) {
	called := false
	handler := ParseBody[validReq](4, func(w http.ResponseWriter, r *http.Request, req *validReq) {
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

func TestParseBody_ValidationFailed(t *testing.T) {
	called := false
	handler := ParseBody[invalidReq](1<<20, func(w http.ResponseWriter, r *http.Request, req *invalidReq) {
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

func TestParsePath(t *testing.T) {
	called := false
	router := chi.NewRouter()
	router.Get("/posts/{slug}", ParsePath("slug", func(w http.ResponseWriter, r *http.Request, slug string) {
		called = true
		if slug != "hello" {
			t.Fatalf("slug = %q", slug)
		}
		w.WriteHeader(http.StatusOK)
	}))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/posts/hello", nil))

	if !called {
		t.Fatal("next was not called")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestParsePositiveInt64PathRejectsInvalidID(t *testing.T) {
	called := false
	router := chi.NewRouter()
	router.Delete("/comments/{id}", ParsePositiveInt64Path("id", func(http.ResponseWriter, *http.Request, int64) {
		called = true
	}))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, httptest.NewRequest(http.MethodDelete, "/comments/invalid", nil))

	if called {
		t.Fatal("next was called for invalid id")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestParsePaginationDefaults(t *testing.T) {
	called := false
	handler := ParsePagination(func(w http.ResponseWriter, r *http.Request, pagination Pagination) {
		called = true
		if pagination.Page != 1 || pagination.PageSize != 20 {
			t.Fatalf("pagination = %+v", pagination)
		}
		w.WriteHeader(http.StatusOK)
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/posts", nil))

	if !called {
		t.Fatal("next was not called")
	}
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
}

func TestParsePaginationRejectsInvalidValue(t *testing.T) {
	called := false
	handler := ParsePagination(func(http.ResponseWriter, *http.Request, Pagination) {
		called = true
	})

	w := httptest.NewRecorder()
	handler.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/posts?page_size=101", nil))

	if called {
		t.Fatal("next was called for invalid pagination")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}
