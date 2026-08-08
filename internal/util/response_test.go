package util

import (
	"context"
	"encoding/json"
	"errors"
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
	Reply(context.Background(), w, map[string]string{"key": "val"}, CodeSuccess, nil)

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
	Reply(context.Background(), w, map[string]int{"id": 1}, CodeCreated, nil)

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
	Reply(context.Background(), w, nil, CodeNoContent, nil)

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
	Reply(context.Background(), w, nil, CodeBadRequest, NewAppError("参数错误", nil))

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
	Reply(context.Background(), w, nil, CodeUnauthorized, NewAppError("login required", nil))

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
	Reply(context.Background(), w, nil, CodeForbidden, NewAppError("insufficient permissions", nil))

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
	Reply(context.Background(), w, nil, CodeNotFound, NewAppError("not found", nil))

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
	Reply(context.Background(), w, nil, CodeValidationFailed, NewAppError("username taken", nil))

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
	Reply(context.Background(), w, nil, CodeInternalError, NewAppError("server error", errors.New("database password leaked")))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d", w.Code)
	}
	code, msg, _ := parseResp(t, w)
	if code != 500 || msg != "server error" {
		t.Fatalf("code=%d msg=%q", code, msg)
	}
}

func TestAppErrorRetainsCause(t *testing.T) {
	cause := errors.New("database unavailable")
	err := NewAppError("failed to load post", cause)

	if !errors.Is(err, cause) {
		t.Fatal("AppError does not retain its cause")
	}
}

func TestReply_UnknownErrorUsesGenericInternalResponse(t *testing.T) {
	w := httptest.NewRecorder()
	Reply(context.Background(), w, nil, CodeNotFound, errors.New("database password leaked"))

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	code, msg, data := parseResp(t, w)
	if code != int(CodeInternalError) || msg != "internal server error" || data != nil {
		t.Fatalf("code=%d msg=%q data=%v", code, msg, data)
	}
}
