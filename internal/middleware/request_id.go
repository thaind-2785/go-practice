package middleware

import (
	"context"
	"crypto/rand"
	"net/http"
)

// RequestID gives every request a short random ID, stores it in the context
// and returns it in the X-Request-ID header so a client error can be matched to a log line.
func RequestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := rand.Text()[:10]
		w.Header().Set("X-Request-ID", id)

		ctx := context.WithValue(r.Context(), requestIDKey, id)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
