package middleware

import (
	"context"
	"net/http"
)

// Middleware wraps a handler to run code before and/or after it.
type Middleware func(http.Handler) http.Handler

// Chain wraps h with mws so that the first middleware is the outermost one:
// Chain(h, A, B, C) is the same as A(B(C(h))), and a request passes A → B → C → h.
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// ctxKey is unexported so no other package can read or overwrite these context values by accident.
type ctxKey int

const (
	requestIDKey ctxKey = iota
	userKey
)

// RequestIDFromContext returns the ID set by the RequestID middleware, or "" if there is none.
func RequestIDFromContext(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// UserFromContext returns the username set by the BasicAuth middleware.
func UserFromContext(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(userKey).(string)
	return user, ok
}
