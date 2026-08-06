package web

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
)

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Validatable is implemented by request DTOs that define their API contract.
type Validatable interface {
	Validate() error
}

func WriteJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

// DecodeJSON parses one JSON value and writes a 400 response when parsing
// fails.
func DecodeJSON(w http.ResponseWriter, r *http.Request, v any, maxBytes int64) bool {
	if err := decodeJSON(w, r, v, maxBytes); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", "invalid request body")
		return false
	}
	return true
}

// Validate validates a request DTO and writes a 400 response when invalid.
func Validate(w http.ResponseWriter, request Validatable) bool {
	if err := request.Validate(); err != nil {
		WriteError(w, http.StatusBadRequest, "invalid_request", err.Error())
		return false
	}
	return true
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any, maxBytes int64) error {
	dec := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxBytes))
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	// Ensure the body contains exactly one JSON value.
	var extra any
	if err := dec.Decode(&extra); err != nil {
		if errors.Is(err, io.EOF) {
			return nil
		}
		return err // preserves *http.MaxBytesError and other real errors
	}
	return errors.New("request body must contain a single JSON value")
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, struct {
		Error APIError `json:"error"`
	}{Error: APIError{Code: code, Message: message}})
}
