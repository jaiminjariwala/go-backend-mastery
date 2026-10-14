// adds logging around a handler
package middleware

import (
	"log"
	"net/http"
	"time"
)

// Logging does shared work before and after the requested job.
func Logging(logger *log.Logger, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()  // Before: start the timer.
		next.ServeHTTP(w, r) // Let the handler do its job.
		// After: record what was requested and how long it took.
		// Do not log private bodies or query parameters.
		logger.Printf("method=%q path=%q duration=%s", r.Method, r.URL.Path, time.Since(start))
	})
}