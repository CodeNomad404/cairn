package middleware

import (
	"context"
	"net/http"

	"cairn.cn/internal/auth"
	authlogic "cairn.cn/internal/auth/logic"
	"cairn.cn/internal/httpx"
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

// WithUser 返回设置了指定用户的子 context，用于测试。
func WithUser(ctx context.Context, user *auth.User) context.Context {
	return context.WithValue(ctx, userKey, user)
}

// RequireUser 拒绝未认证用户的请求。
func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetUser(r.Context()) == nil {
			httpx.Reply(r.Context(), w, nil, httpx.CodeUnauthorized, httpx.NewAppError("login required", nil))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// RequireRole 拒绝不具备所需角色的已认证用户。
func RequireRole(role string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user := GetUser(r.Context())
			if user == nil {
				httpx.Reply(r.Context(), w, nil, httpx.CodeUnauthorized, httpx.NewAppError("login required", nil))
				return
			}
			if user.Role != role {
				httpx.Reply(r.Context(), w, nil, httpx.CodeForbidden, httpx.NewAppError("insufficient permissions", nil))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
