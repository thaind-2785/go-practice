package middleware

import (
	"log"
	"net/http"
	"runtime/debug"

	"github.com/thaind-2785/go-practice/internal/utils"
)

// Recover turns a panic in any inner handler into a JSON 500 response instead of a dropped connection.
func Recover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec == nil {
				return
			}
			// net/http panics with ErrAbortHandler on purpose to abort a response; let it through.
			if rec == http.ErrAbortHandler {
				panic(rec)
			}

			log.Printf("[%s] panic: %v\n%s", RequestIDFromContext(r.Context()), rec, debug.Stack())
			utils.WriteError(w, utils.NewInternal("internal server error"))
		}()

		next.ServeHTTP(w, r)
	})
}
