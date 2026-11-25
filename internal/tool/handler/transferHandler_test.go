package handler

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"cairn.cn/internal/auth"
	authmid "cairn.cn/internal/auth/middleware"
	"github.com/go-chi/chi/v5"
)

type unreadTransferBody struct{ t *testing.T }

func (b unreadTransferBody) Read([]byte) (int, error) {
	b.t.Fatal("read body before authorization")
	return 0, io.EOF
}
func (b unreadTransferBody) Close() error { return nil }

func TestTransferRoutesAuthorizeBeforeParsing(t *testing.T) {
	routes := [][2]string{
		{"POST", "transfer-channel/create"}, {"GET", "transfer-channel/list"}, {"POST", "transfer-channel/update"},
		{"POST", "transfer-channel/delete"}, {"POST", "transfer/create"}, {"POST", "transfer/upload"},
		{"GET", "transfer/list"}, {"GET", "transfer/get"}, {"GET", "transfer/download"},
		{"POST", "transfer/delete"}, {"GET", "transfer-usage/get"},
	}
	router := chi.NewRouter()
	NewTransferHandler(nil).RegisterRoutes(router)
	for _, route := range routes {
		for _, role := range []string{"", auth.RoleUser} {
			t.Run(route[1]+"/"+role, func(t *testing.T) {
				req := httptest.NewRequest(route[0], "/tool/"+route[1], unreadTransferBody{t})
				want := http.StatusUnauthorized
				if role != "" {
					req = req.WithContext(authmid.WithPrincipal(req.Context(), &auth.Principal{ID: 1, Role: role}))
					want = http.StatusForbidden
				}
				response := httptest.NewRecorder()
				router.ServeHTTP(response, req)
				if response.Code != want {
					t.Fatalf("status=%d want=%d", response.Code, want)
				}
				if response.Header().Get("Cache-Control") != "no-store" {
					t.Fatal("private response may be cached")
				}
			})
		}
	}
}

func TestTransferInvalidInputNeverReachesLogic(t *testing.T) {
	router := chi.NewRouter()
	NewTransferHandler(nil).RegisterRoutes(router)
	cases := [][3]string{
		{"POST", "transfer-channel/create", `{"name":"  "}`},
		{"POST", "transfer-channel/update", `{"id":0,"name":"valid"}`},
		{"POST", "transfer-channel/delete", `{"id":1,"unknown":true}`},
		{"POST", "transfer/create", `{"channel_id":1,"text":"\u0000"}`},
		{"POST", "transfer/create", `{"channel_id":1,"text":"  \n"}`},
		{"POST", "transfer/create", `{"channel_id":1,"text":"` + strings.Repeat("文", 35000) + `"}`},
		{"GET", "transfer/list?channel_id=1&kind=html", ""},
		{"GET", "transfer/get?id=-1", ""},
		{"GET", "transfer/download?id=abc", ""},
	}
	for _, tt := range cases {
		t.Run(tt[1], func(t *testing.T) {
			req := httptest.NewRequest(tt[0], "/tool/"+tt[1], strings.NewReader(tt[2]))
			req.Header.Set("Content-Type", "application/json")
			req = req.WithContext(authmid.WithPrincipal(req.Context(), &auth.Principal{ID: 1, Role: auth.RoleAdmin}))
			response := httptest.NewRecorder()
			router.ServeHTTP(response, req)
			if response.Code != 400 {
				t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
			}
		})
	}
}
