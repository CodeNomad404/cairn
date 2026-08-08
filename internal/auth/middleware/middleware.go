package middleware

import (
	"context"
	"net/http"

	"cairn.cn/internal/auth"
	authlogic "cairn.cn/internal/auth/logic"
	"cairn.cn/internal/util"
)

type contextKey string

const userKey contextKey = "user"

// LoadUser 在每个请求中注入用户信息
func LoadUser(authLogic *authlogic.AuthLogic) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(auth.SessionCookie)
			if err == nil && cookie.Value != "" {
				user, err := authLogic.GetUserByToken(r.Context(), cookie.Value)
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
			util.Reply(r.Context(), w, nil, util.CodeUnauthorized, util.NewAppError("login required", nil))
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
				util.Reply(r.Context(), w, nil, util.CodeUnauthorized, util.NewAppError("login required", nil))
				return
			}
			if user.Role != role {
				util.Reply(r.Context(), w, nil, util.CodeForbidden, util.NewAppError("insufficient permissions", nil))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
