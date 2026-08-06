package web

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type invalidRequest struct{}

func (invalidRequest) Validate() error {
	return errors.New("a field is required")
}

func TestDecodeJSON_Valid(t *testing.T) {
	body := strings.NewReader(`{"slug":"hello","title":"Hello","content_md":"# Hi"}`)
	r := httptest.NewRequest("POST", "/", body)
	w := httptest.NewRecorder()

	var input struct {
		Slug      string `json:"slug"`
		Title     string `json:"title"`
		ContentMD string `json:"content_md"`
	}
	if !DecodeJSON(w, r, &input, 1<<20) {
		t.Fatal("expected valid JSON to be accepted")
	}
	if input.Slug != "hello" || input.Title != "Hello" || input.ContentMD != "# Hi" {
		t.Fatalf("input = %+v", input)
	}
}

func TestDecodeJSON_InvalidJSON(t *testing.T) {
	body := strings.NewReader(`not-json`)
	r := httptest.NewRequest("POST", "/", body)
	w := httptest.NewRecorder()

	var input struct {
		Slug string `json:"slug"`
	}
	if DecodeJSON(w, r, &input, 1<<20) {
		t.Fatal("expected invalid JSON to be rejected")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestValidateWritesBadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	if Validate(w, invalidRequest{}) {
		t.Fatal("expected invalid request to be rejected")
	}
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestDecodeJSON_RejectsUnknownFields(t *testing.T) {
	body := strings.NewReader(`{"slug":"hello","unexpected":true}`)
	r := httptest.NewRequest("POST", "/", body)
	w := httptest.NewRecorder()

	var input struct {
		Slug string `json:"slug"`
	}
	if DecodeJSON(w, r, &input, 1<<20) {
		t.Fatal("expected unknown field to be rejected")
	}
}

func TestDecodeJSON_TrailingGarbage(t *testing.T) {
	body := strings.NewReader(`{"slug":"a"} {"unexpected":"second value"}`)
	r := httptest.NewRequest("POST", "/", body)
	w := httptest.NewRecorder()

	var input struct {
		Slug string `json:"slug"`
	}
	if DecodeJSON(w, r, &input, 1<<20) {
		t.Fatal("expected trailing garbage to be rejected")
	}
}

func TestDecodeJSON_TrailingExceedsLimit(t *testing.T) {
	// Valid first JSON value, then trailing whitespace large enough that the
	// decoder exhausts its buffer during the trailing scan and calls Read again,
	// hitting the MaxBytesReader limit.
	body := strings.NewReader(`{"slug":"a"}` + strings.Repeat(" ", 2000))
	r := httptest.NewRequest("POST", "/", body)
	w := httptest.NewRecorder()

	var input struct {
		Slug string `json:"slug"`
	}
	if DecodeJSON(w, r, &input, 512) {
		t.Fatal("expected trailing data exceeding limit to be rejected")
	}
}

func TestDecodeJSON_BodyTooLarge(t *testing.T) {
	body := strings.NewReader(`{"slug":"` + strings.Repeat("x", 2<<20) + `"}`)
	r := httptest.NewRequest("POST", "/", body)
	w := httptest.NewRecorder()

	var input struct {
		Slug string `json:"slug"`
	}
	if DecodeJSON(w, r, &input, 4) { // tiny limit
		t.Fatal("expected body exceeding limit to be rejected")
	}
}
