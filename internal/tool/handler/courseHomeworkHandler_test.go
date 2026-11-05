package handler

import (
	"net/http/httptest"
	"strings"
	"testing"

	"cairn.cn/internal/auth"
	authmid "cairn.cn/internal/auth/middleware"
	"github.com/go-chi/chi/v5"
)

func TestHomeworkAuthenticationAndInput(t *testing.T) {
	r := chi.NewRouter()
	NewHomeworkHandler(nil).RegisterRoutes(r)
	RegisterMISBind(r, nil)
	for _, tc := range []struct {
		method, path, body string
		authenticated      bool
		want               int
	}{
		{"POST", "/tool/mis-account/bind", "{", false, 401},
		{"POST", "/tool/course-homework/sync", "{", false, 401},
		{"GET", "/tool/course-homework/get?id=1", "", false, 401},
		{"GET", "/tool/course-homework/list", "", false, 401},
		{"POST", "/tool/mis-account/bind", `{"student_id":"s","password":"p","user_id":1}`, true, 400},
		{"POST", "/tool/course-homework/sync", `{"user_id":1}`, true, 400},
		{"POST", "/tool/mis-account/bind", `{"student_id":"s","password":""}`, true, 400},
		{"GET", "/tool/course-homework/get?id=-1", "", true, 400},
		{"GET", "/tool/course-homework/list?status=invalid", "", true, 400},
		{"GET", "/tool/course-homework/list?page_size=101", "", true, 400},
	} {
		t.Run(tc.method+tc.path+tc.body, func(t *testing.T) {
			req := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			if tc.authenticated {
				req = req.WithContext(authmid.WithPrincipal(req.Context(), &auth.Principal{ID: 7}))
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			if w.Code != tc.want {
				t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
			}
		})
	}
}
