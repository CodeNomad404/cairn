package middleware

import (
	"net"
	"net/http"
	"net/url"
	"strings"
)

// SecurityHeaders 为每个响应添加保守的浏览器安全头。
func SecurityHeaders(production bool) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'")
			w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
			w.Header().Set("X-Content-Type-Options", "nosniff")
			w.Header().Set("X-Frame-Options", "DENY")
			w.Header().Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
			if production {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			}
			next.ServeHTTP(w, r)
		})
	}
}

// SameOrigin 拒绝由其他源发起的浏览器写请求。
func SameOrigin() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if isSafeMethod(r.Method) || isSameOrigin(r) {
				next.ServeHTTP(w, r)
				return
			}
			http.Error(w, "cross-origin request rejected", http.StatusForbidden)
		})
	}
}

// TrustedProxyRealIP 只接受来自回环代理的转发客户端 IP。
func TrustedProxyRealIP() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			host, _, err := net.SplitHostPort(r.RemoteAddr)
			if err == nil {
				proxyIP := net.ParseIP(host)
				if proxyIP != nil && proxyIP.IsLoopback() {
					if forwarded := firstForwardedIP(r.Header.Get("X-Forwarded-For")); forwarded != "" {
						r.RemoteAddr = forwarded
					} else if realIP := strings.TrimSpace(r.Header.Get("X-Real-IP")); realIP != "" {
						r.RemoteAddr = realIP
					}
				}
			}
			next.ServeHTTP(w, r)
		})
	}
}

func firstForwardedIP(value string) string {
	if value == "" {
		return ""
	}
	return strings.TrimSpace(strings.Split(value, ",")[0])
}

func isSafeMethod(method string) bool {
	return method == http.MethodGet || method == http.MethodHead || method == http.MethodOptions
}

func isSameOrigin(r *http.Request) bool {
	switch r.Header.Get("Sec-Fetch-Site") {
	case "cross-site":
		return false
	case "same-origin", "none":
		return true
	}

	if origin := r.Header.Get("Origin"); origin != "" {
		return urlHost(origin) == r.Host
	}
	if referer := r.Header.Get("Referer"); referer != "" {
		return urlHost(referer) == r.Host
	}

	// 非浏览器客户端不会自动携带浏览器环境凭据。
	return true
}

func urlHost(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	return parsed.Host
}
