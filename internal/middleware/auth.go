package middleware

import (
	"context"
	"crypto/subtle"
	"log"
	"net/http"

	"github.com/thaind-2785/go-practice/internal/utils"
)

func BasicAuth(users map[string]string) Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			user, pass, ok := r.BasicAuth()
			if !ok || !validCredentials(users, user, pass) {
				if ok {
					log.Printf("[%s] auth failed for user %q", RequestIDFromContext(r.Context()), user)
				}
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
	return ok && subtle.ConstantTimeCompare([]byte(pass), []byte(expected)) == 1
}
