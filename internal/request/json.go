package request

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"cairn.cn/internal/response"
)

// Validatable is implemented by request DTOs that define their own API contract.
type Validatable interface {
	Validate() error
}

// WithValidatedJSON returns an http.HandlerFunc that decodes the request body
// into a value of type T, validates it, and calls next on success.
// On failure it writes a standard error response and does not call next.
func WithValidatedJSON[T Validatable](maxBytes int64, next func(http.ResponseWriter, *http.Request, *T)) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req T
		if err := decodeJSON(w, r, &req, maxBytes); err != nil {
			response.Reply(r.Context(), w, nil, response.CodeBadRequest, "参数错误")
			return
		}
		if err := req.Validate(); err != nil {
			response.Reply(r.Context(), w, nil, response.CodeBadRequest, "参数错误")
			return
		}
		next(w, r, &req)
	}
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
