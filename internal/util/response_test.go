package util

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func parseResp(t *testing.T, w *httptest.ResponseRecorder) (int, string, any) {
	t.Helper()
	var resp struct {
		Code    int    `json:"code"`
		Message string `json:"message,omitempty"`
		Data    any    `json:"data,omitempty"`
	}
	if err := json.NewDecoder(w.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	return resp.Code, resp.Message, resp.Data
}

func TestReply_Success(t *testing.T) {
	w := httptest.NewRecorder()
	Reply(context.Background(), w, map[string]string{"key": "val"}, CodeSuccess, "")

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	if got := w.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("Content-Type = %q", got)
	}
	code, msg, data := parseResp(t, w)
	if code != 0 || msg != "" || data == nil {
		t.Fatalf("code=%d msg=%q data=%v", code, msg, data)
	}
}

func TestReply_Created(t *testing.T) {
	w := httptest.NewRecorder()
	Reply(context.Background(), w, map[string]int{"id": 1}, CodeCreated, "")

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d", w.Code)
	}
	code, msg, data := parseResp(t, w)
	if code != 1 || msg != "" || data == nil {
		t.Fatalf("code=%d msg=%q data=%v", code, msg, data)
	}
}

func TestReply_NoContent(t *testing.T) {
	w := httptest.NewRecorder()
	Reply(context.Background(), w, nil, CodeNoContent, "")

	if w.Code != http.StatusNoContent {
		t.Fatalf("status = %d", w.Code)
	}
	// NoContent should have no body.
	if w.Body.Len() != 0 {
		t.Fatalf("body should be empty, got %q", w.Body.String())
	}
}

func TestReply_BadRequest(t *testing.T) {
	w := httptest.NewRecorder()
	Reply(context.Background(), w, nil, CodeBadRequest, "参数错误")

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d", w.Code)
	}
	code, msg, data := parseResp(t, w)
	if code != 400 || msg != "参数错误" || data != nil {
		t.Fatalf("code=%d msg=%q data=%v", code, msg, data)
	}
}

func TestReply_Unauthorized(t *testing.T) {
	w := httptest.NewRecorder()
	Reply(context.Background(), w, nil, CodeUnauthorized, "login required")

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d", w.Code)
	}
	code, msg, _ := parseResp(t, w)
	if code != 401 || msg != "login required" {
		t.Fatalf("code=%d msg=%q", code, msg)
	}
}

func TestReply_Forbidden(t *testing.T) {
	w := httptest.NewRecorder()
	Reply(context.Background(), w, nil, CodeForbidden, "insufficient permissions")

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d", w.Code)
	}
	code, msg, _ := parseResp(t, w)
	if code != 403 {
		t.Fatalf("code=%d msg=%q", code, msg)
	}
}

func TestReply_NotFound(t *testing.T) {
	w := httptest.NewRecorder()
	Reply(context.Background(), w, nil, CodeNotFound, "not found")

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d", w.Code)
	}
	code, msg, _ := parseResp(t, w)
	if code != 404 {
		t.Fatalf("code=%d msg=%q", code, msg)
	}
}

func TestReply_ValidationFailed(t *testing.T) {
	w := httptest.NewRecorder()
	Reply(context.Background(), w, nil, CodeValidationFailed, "username taken")

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status = %d, want 422", w.Code)
	}
	code, msg, _ := parseResp(t, w)
	if code != 422 {
		t.Fatalf("code=%d msg=%q", code, msg)
	}
}

func TestReply_InternalError(t *testing.T) {
	w := httptest.NewRecorder()
	Reply(context.Background(), w, nil, CodeInternalError, "server error")

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
	code, msg, _ := parseResp(t, w)
	if code != 500 {
		t.Fatalf("code=%d msg=%q", code, msg)
	}
}
