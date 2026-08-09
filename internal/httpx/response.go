package httpx

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
)

// Code 是机器可读的响应业务码。
type Code int

// 全部响应业务码。
const (
	CodeSuccess          Code = 0   // 成功
	CodeCreated          Code = 1   // 创建成功
	CodeNoContent        Code = 2   // 无内容
	CodeBadRequest       Code = 400 // 请求参数错误
	CodeUnauthorized     Code = 401 // 未认证
	CodeForbidden        Code = 403 // 无权限
	CodeNotFound         Code = 404 // 资源不存在
	CodeValidationFailed Code = 422 // 验证失败
	CodeInternalError    Code = 500 // 内部错误
)

// AppError 保存可安全返回给客户端的消息，同时保留原始错误原因。
type AppError struct {
	Message string
	cause   error
}

func NewAppError(message string, cause error) *AppError {
	return &AppError{Message: message, cause: cause}
}

func (e *AppError) Error() string {
	if e.cause == nil {
		return e.Message
	}
	return fmt.Sprintf("%s: %v", e.Message, e.cause)
}

func (e *AppError) Unwrap() error {
	return e.cause
}

// HTTPStatus 将业务码映射为 HTTP 状态码。
func (c Code) HTTPStatus() int {
	switch c {
	case CodeSuccess:
		return http.StatusOK
	case CodeCreated:
		return http.StatusCreated
	case CodeNoContent:
		return http.StatusNoContent
	case CodeBadRequest:
		return http.StatusBadRequest
	case CodeUnauthorized:
		return http.StatusUnauthorized
	case CodeForbidden:
		return http.StatusForbidden
	case CodeNotFound:
		return http.StatusNotFound
	case CodeValidationFailed:
		return http.StatusUnprocessableEntity
	default:
		return http.StatusInternalServerError
	}
}

// body 是标准 JSON 响应信封。
type body struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// Reply 写入标准 JSON 响应。
func Reply(ctx context.Context, w http.ResponseWriter, data any, code Code, err error) {
	_ = ctx

	if err != nil {
		var appErr *AppError
		if !errors.As(err, &appErr) {
			code = CodeInternalError
			appErr = NewAppError("internal server error", nil)
		}
		if code == CodeSuccess || code == CodeCreated || code == CodeNoContent {
			code = CodeInternalError
		}
		writeJSON(w, code.HTTPStatus(), body{Code: int(code), Message: appErr.Message})
		return
	}

	if code == CodeNoContent {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	if code == CodeSuccess || code == CodeCreated {
		if data == nil {
			writeJSON(w, code.HTTPStatus(), body{Code: int(code)})
			return
		}
		writeJSON(w, code.HTTPStatus(), body{Code: int(code), Data: data})
		return
	}

	writeJSON(w, http.StatusInternalServerError, body{Code: int(CodeInternalError), Message: "internal server error"})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
