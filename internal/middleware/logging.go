package middleware

import (
	"log"
	"net/http"
	"time"
)

// statusRecorder remembers the status code a handler writes, which http.ResponseWriter does not expose.
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Unwrap lets http.ResponseController reach the original writer.
func (r *statusRecorder) Unwrap() http.ResponseWriter {
	return r.ResponseWriter
}

// Logging logs request ID, method, URL, status, duration and authenticated user of each request.
func Logging(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		// A handler that writes a body without calling WriteHeader gets 200.
		rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}

		next.ServeHTTP(rec, r)

		user, _ := UserFromContext(r.Context())
		log.Printf("[%s] %s %s %d %s user=%s",
			RequestIDFromContext(r.Context()), r.Method, r.URL.RequestURI(), rec.status,
			time.Since(start).Round(time.Microsecond), user)
	})
}
