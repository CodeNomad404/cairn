package httpx

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

// Pagination contains validated page query parameters.
type Pagination struct {
	Page     int
	PageSize int
}

// ParseBody decodes and validates a JSON request body into a value of type T.
// It enforces the body size limit, rejects unknown fields and trailing data,
// and calls Validate on the decoded value. On failure it returns an AppError
// whose Cause is the underlying decode/validate error; it never writes a
// response. The caller decides the HTTP status and response body.
func ParseBody[T Validatable](w http.ResponseWriter, r *http.Request, maxBytes int64) (*T, error) {
	var req T
	if err := decodeJSON(w, r, &req, maxBytes); err != nil {
		return nil, NewAppError(err.Error(), err)
	}
	if err := req.Validate(); err != nil {
		return nil, NewAppError(err.Error(), err)
	}
	return &req, nil
}

// ParsePath returns the trimmed non-empty value of a chi path parameter.
func ParsePath(r *http.Request, name string) (string, error) {
	value := strings.TrimSpace(chi.URLParam(r, name))
	if value == "" {
		return "", NewAppError("参数错误", nil)
	}
	return value, nil
}

// ParsePositiveInt64Path parses a positive int64 path parameter.
func ParsePositiveInt64Path(r *http.Request, name string) (int64, error) {
	value, err := ParsePath(r, name)
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id < 1 {
		return 0, NewAppError("参数错误", err)
	}
	return id, nil
}

// ParsePagination parses page and page_size, applying the standard defaults
// (page=1, page_size=20) and limits (page>=1, 1<=page_size<=100).
func ParsePagination(r *http.Request) (Pagination, error) {
	page, err := queryInt(r, "page", 1, 1, 0)
	if err != nil {
		return Pagination{}, err
	}
	pageSize, err := queryInt(r, "page_size", 20, 1, 100)
	if err != nil {
		return Pagination{}, err
	}
	return Pagination{Page: page, PageSize: pageSize}, nil
}

func queryInt(r *http.Request, name string, defaultValue, minValue, maxValue int) (int, error) {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return defaultValue, nil
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minValue || maxValue > 0 && value > maxValue {
		return 0, NewAppError("参数错误", err)
	}
	return value, nil
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
