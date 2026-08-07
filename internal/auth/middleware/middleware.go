package middleware

import (
	"context"
	"net/http"

	"cairn.cn/internal/auth"
	authlogic "cairn.cn/internal/auth/logic"
	"cairn.cn/internal/response"
	"cairn.cn/internal/svc"
)

type contextKey string

const userKey contextKey = "user"

// LoadUser 在每个请求中注入用户信息
func LoadUser(svcCtx *svc.ServiceContext) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(auth.SessionCookie)
			if err == nil && cookie.Value != "" {
				l := authlogic.NewAuthLogic(r.Context(), svcCtx)
				user, err := l.GetUserByToken(cookie.Value)
				if err == nil {
					ctx := context.WithValue(r.Context(), userKey, user)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetUser 从 context 中获取当前登录用户，未登录返回 nil
func GetUser(ctx context.Context) *auth.User {
	if u, ok := ctx.Value(userKey).(*auth.User); ok {
		return u
	}
	return nil
}

// WithUser returns a child context with the given user set, for testing.
func WithUser(ctx context.Context, user *auth.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

// RequireUser rejects requests without an authenticated user.
func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetUser(r.Context()) == nil {
			response.Reply(r.Context(), w, nil, response.CodeUnauthorized, "login required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole rejects authenticated users without the required role.
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r.Context())
			if user == nil {
				response.Reply(r.Context(), w, nil, response.CodeUnauthorized, "login required")
				return
			}
			if user.Role != role {
				response.Reply(r.Context(), w, nil, response.CodeForbidden, "insufficient permissions")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
