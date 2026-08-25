package httpui

import (
	"net/http"
	"strings"
	"time"
)

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("Content-Security-Policy", "default-src 'self'; style-src 'self'; script-src 'self'")
		w.Header().Set("Cache-Control", "no-store")
		started := time.Now()
		next.ServeHTTP(w, r)
		_ = started
	})
}

func requestID(r *http.Request) string {
	value := strings.TrimSpace(r.Header.Get("X-Request-ID"))
	if len(value) > 80 {
		return value[:80]
	}
	return value
}
