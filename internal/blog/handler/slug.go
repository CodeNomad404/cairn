package handler

import (
	"net/http"
	"net/url"

	"github.com/go-chi/chi/v5"
)

func slugParam(r *http.Request) string {
	slug := chi.URLParam(r, "slug")
	if decoded, err := url.PathUnescape(slug); err == nil {
		return decoded
	}
	return slug
}
