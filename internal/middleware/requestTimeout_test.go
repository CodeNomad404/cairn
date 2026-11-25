package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

type transferDeadlineRecorder struct {
	*httptest.ResponseRecorder
	read  time.Time
	write time.Time
}

func (r *transferDeadlineRecorder) SetReadDeadline(deadline time.Time) error {
	r.read = deadline
	return nil
}
func (r *transferDeadlineRecorder) SetWriteDeadline(deadline time.Time) error {
	r.write = deadline
	return nil
}

func TestOnlyTransferStreamsExtendRequestAndConnectionTimeouts(t *testing.T) {
	tests := []struct {
		method, path string
		long         bool
	}{
		{"POST", "/api/v1/tool/transfer/upload", true},
		{"GET", "/api/v1/tool/transfer/download", true},
		{"GET", "/api/v1/tool/transfer/list", false},
		{"POST", "/api/v1/tool/transfer/create", false},
		{"POST", "/api/v1/tool/transfer/download", false},
		{"GET", "/healthz", false},
	}
	for _, tt := range tests {
		t.Run(tt.method+tt.path, func(t *testing.T) {
			w := &transferDeadlineRecorder{ResponseRecorder: httptest.NewRecorder()}
			next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				deadline, ok := r.Context().Deadline()
				if !ok {
					t.Fatal("missing context timeout")
				}
				want := 30 * time.Second
				if tt.long {
					want = 10 * time.Minute
				}
				if remaining := time.Until(deadline); remaining < want-time.Second || remaining > want {
					t.Fatalf("deadline=%v want=%v", remaining, want)
				}
				w.WriteHeader(204)
			})
			RequestTimeout(next).ServeHTTP(w, httptest.NewRequest(tt.method, tt.path, nil))
			if tt.long {
				if w.read.IsZero() || w.write.IsZero() {
					t.Fatal("connection deadlines not extended")
				}
			} else if !w.read.IsZero() || !w.write.IsZero() {
				t.Fatal("ordinary connection deadline changed")
			}
		})
	}
}
