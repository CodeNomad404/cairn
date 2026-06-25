package web

import (
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"
)

type rateLimitEntry struct {
	count   int
	resetAt time.Time
}

// RateLimit limits requests per client IP within a fixed time window.
func RateLimit(limit int, window time.Duration) func(http.Handler) http.Handler {
	var mu sync.Mutex
	entries := make(map[string]rateLimitEntry)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			now := time.Now()
			key := clientIP(r)

			mu.Lock()
			if len(entries) > 10_000 {
				for client, existing := range entries {
					if now.After(existing.resetAt) {
						delete(entries, client)
					}
				}
			}
			entry := entries[key]
			if entry.resetAt.IsZero() || now.After(entry.resetAt) {
				entry = rateLimitEntry{resetAt: now.Add(window)}
			}
			entry.count++
			entries[key] = entry
			mu.Unlock()

			if entry.count > limit {
				w.Header().Set("Retry-After", retryAfterSeconds(now, entry.resetAt))
				http.Error(w, "too many requests", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func clientIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

func retryAfterSeconds(now, resetAt time.Time) string {
	seconds := int(resetAt.Sub(now).Seconds())
	if seconds < 1 {
		seconds = 1
	}
	return strconv.Itoa(seconds)
}
