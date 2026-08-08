package util

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-chi/chi/v5"
)

// Validatable is implemented by request DTOs that define their own API contract.
type Validatable interface {
	Validate() error
}

// ParseBody returns an http.HandlerFunc that decodes the request body
// into a value of type T, validates it, and calls next on success.
// On failure it writes a standard error response and does not call next.
func ParseBody[T Validatable](maxBytes int64, next func(http.ResponseWriter, *http.Request, *T)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		req, ok := parseBody[T](w, r, maxBytes)
		if ok {
			next(w, r, req)
		}
	}
}

// ParsePathAndBody parses a string path parameter and a validated JSON body.
func ParsePathAndBody[T Validatable](name string, maxBytes int64, next func(http.ResponseWriter, *http.Request, string, *T)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		value, ok := pathValue(w, r, name)
		if !ok {
			return
		}
		req, ok := parseBody[T](w, r, maxBytes)
		if ok {
			next(w, r, value, req)
		}
	}
}

// ParsePath passes a non-empty string path parameter to next.
func ParsePath(name string, next func(http.ResponseWriter, *http.Request, string)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		value, ok := pathValue(w, r, name)
		if ok {
			next(w, r, value)
		}
	}
}

// ParsePositiveInt64Path parses a positive int64 path parameter and passes it to next.
func ParsePositiveInt64Path(name string, next func(http.ResponseWriter, *http.Request, int64)) http.HandlerFunc {
	return ParsePath(name, func(w http.ResponseWriter, r *http.Request, value string) {
		id, err := strconv.ParseInt(value, 10, 64)
		if err != nil || id < 1 {
			Reply(r.Context(), w, nil, CodeBadRequest, "参数错误")
			return
		}
		next(w, r, id)
	})
}

// Pagination contains validated page query parameters.
type Pagination struct {
	Page     int
	PageSize int
}

// ParsePagination parses page and page_size, applying the standard defaults and limits.
func ParsePagination(next func(http.ResponseWriter, *http.Request, Pagination)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		page, ok := queryInt(r, "page", 1, 1, 0)
		if !ok {
			Reply(r.Context(), w, nil, CodeBadRequest, "参数错误")
			return
		}
		pageSize, ok := queryInt(r, "page_size", 20, 1, 100)
		if !ok {
			Reply(r.Context(), w, nil, CodeBadRequest, "参数错误")
			return
		}
		next(w, r, Pagination{Page: page, PageSize: pageSize})
	}
}

func parseBody[T Validatable](w http.ResponseWriter, r *http.Request, maxBytes int64) (*T, bool) {
	var req T
	if err := decodeJSON(w, r, &req, maxBytes); err != nil {
		Reply(r.Context(), w, nil, CodeBadRequest, err.Error())
		return nil, false
	}
	if err := req.Validate(); err != nil {
		Reply(r.Context(), w, nil, CodeBadRequest, err.Error())
		return nil, false
	}
	return &req, true
}

func pathValue(w http.ResponseWriter, r *http.Request, name string) (string, bool) {
	value := strings.TrimSpace(chi.URLParam(r, name))
	if value == "" {
		Reply(r.Context(), w, nil, CodeBadRequest, "参数错误")
		return "", false
	}
	return value, true
}

func queryInt(r *http.Request, name string, defaultValue, minValue, maxValue int) (int, bool) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return defaultValue, true
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minValue || maxValue > 0 && value > maxValue {
		return 0, false
	}
	return value, true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any, maxBytes int64) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	var extra any
	if err := dec.Decode(&extra); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err
	}
	return errors.New("request body must contain a single JSON value")
}
