package middleware

import (
	"context"
	"net/http"

	"cairn.cn/internal/auth"
	authlogic "cairn.cn/internal/auth/logic"
	"cairn.cn/internal/httpx"
)

type contextKey string

const principalKey contextKey = "principal"

// LoadUser 在每个请求中注入登录身份
func LoadUser(authLogic *authlogic.AuthLogic) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			cookie, err := r.Cookie(auth.SessionCookie)
			if err == nil && cookie.Value != "" {
				principal, err := authLogic.GetPrincipalByToken(r.Context(), cookie.Value)
				if err == nil {
					ctx := context.WithValue(r.Context(), principalKey, principal)
					r = r.WithContext(ctx)
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

// GetPrincipal 从 context 中获取当前登录身份，未登录返回 nil
func GetPrincipal(ctx context.Context) *auth.Principal {
	if p, ok := ctx.Value(principalKey).(*auth.Principal); ok {
		return p
	}
	return nil
}

// WithPrincipal 返回设置了指定身份的 child context，用于测试。
func WithPrincipal(ctx context.Context, principal *auth.Principal) context.Context {
	return context.WithValue(ctx, principalKey, principal)
}

// RequireUser 拒绝未认证用户的请求。
func RequireUser(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if GetPrincipal(r.Context()) == nil {
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
			principal := GetPrincipal(r.Context())
			if principal == nil {
				httpx.Reply(r.Context(), w, nil, httpx.CodeUnauthorized, httpx.NewAppError("login required", nil))
				return
			}
			if principal.Role != role {
				httpx.Reply(r.Context(), w, nil, httpx.CodeForbidden, httpx.NewAppError("insufficient permissions", nil))
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}
