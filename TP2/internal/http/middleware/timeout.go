package middleware

import (
	"net/http"
	"time"
)

// Timeout stops requests that do not complete within the configured duration.
func Timeout(duration time.Duration) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		timeoutHandler := http.TimeoutHandler(next, duration, `{"error":{"code":"timeout","message":"request timed out"}}`)
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json; charset=utf-8")
			timeoutHandler.ServeHTTP(w, r)
		})
	}
}
