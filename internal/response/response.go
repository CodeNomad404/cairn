package response

import (
	"context"
	"encoding/json"
	"net/http"
)

// Code is a machine-readable response code.
type Code int

// All response codes.
const (
	CodeSuccess          Code = 0
	CodeCreated          Code = 1
	CodeNoContent        Code = 2
	CodeBadRequest       Code = 400
	CodeUnauthorized     Code = 401
	CodeForbidden        Code = 403
	CodeNotFound         Code = 404
	CodeValidationFailed Code = 422
	CodeInternalError    Code = 500
)

// HTTPStatus maps a Code to its HTTP status code.
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

// body is the standard JSON response envelope.
type body struct {
	Code    int    `json:"code"`
	Message string `json:"message,omitempty"`
	Data    any    `json:"data,omitempty"`
}

// Reply writes a standard JSON response.
func Reply(ctx context.Context, w http.ResponseWriter, data any, code Code, msg string) {
	_ = ctx

	if code == CodeNoContent {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	httpStatus := code.HTTPStatus()
	if code == CodeSuccess || code == CodeCreated {
		if data == nil {
			writeJSON(w, httpStatus, body{Code: int(code)})
			return
		}
		writeJSON(w, httpStatus, body{Code: int(code), Data: data})
		return
	}

	writeJSON(w, httpStatus, body{Code: int(code), Message: msg})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}
