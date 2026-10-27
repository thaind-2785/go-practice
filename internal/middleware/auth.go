package middleware

import (
	"context"
	"crypto/subtle"
	"log"
	"net/http"

	"github.com/thaind-2785/go-practice/internal/utils"
)

// BasicAuth checks the "Authorization: Basic <base64(user:password)>" header against users
// (username → password) and stores the username in the request context.
func BasicAuth(users map[string]string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok || !validCredentials(users, user, pass) {
				if ok {
					log.Printf("[%s] auth failed for user %q", RequestIDFromContext(r.Context()), user)
				}
				// Tells browsers and clients which auth scheme to use.
				w.Header().Set("WWW-Authenticate", `Basic realm="employees", charset="UTF-8"`)
				utils.WriteError(w, utils.NewUnauthorized("invalid or missing credentials"))
				return
			}

			ctx := context.WithValue(r.Context(), userKey, user)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func validCredentials(users map[string]string, user, pass string) bool {
	expected, ok := users[user]
	// ConstantTimeCompare takes the same time no matter where the strings differ,
	// so the password cannot be guessed character by character from response times.
	return ok && subtle.ConstantTimeCompare([]byte(pass), []byte(expected)) == 1
}
