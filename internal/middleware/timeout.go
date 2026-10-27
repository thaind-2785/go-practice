package middleware

import (
	"context"
	"net/http"
	"time"
)

// Timeout gives every request a deadline. Database calls receive this context, so a query
// still running after d is cancelled and the service turns it into a 503.
func Timeout(d time.Duration) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), d)
			defer cancel()

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}
