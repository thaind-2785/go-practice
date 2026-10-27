package middleware

import (
	"bytes"
	"encoding/json"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// captureLogs redirects the standard logger into a buffer for the duration of the test.
func captureLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	original := log.Writer()
	var buf bytes.Buffer
	log.SetOutput(&buf)
	t.Cleanup(func() { log.SetOutput(original) })
	return &buf
}

// assertJSONError checks that the response is {"error": ..., "code": wantCode} with that status.
func assertJSONError(t *testing.T, rec *httptest.ResponseRecorder, wantCode int) {
	t.Helper()
	if rec.Code != wantCode {
		t.Fatalf("status = %d, want %d", rec.Code, wantCode)
	}
	var body struct {
		Error string `json:"error"`
		Code  int    `json:"code"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not JSON: %q", rec.Body.String())
	}
	if body.Code != wantCode || body.Error == "" {
		t.Errorf("body = %+v, want code %d with a message", body, wantCode)
	}
}

func TestChain_Order(t *testing.T) {
	var order []string
	mark := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	}), mark("A"), mark("B"), mark("C"))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if got := strings.Join(order, ","); got != "A,B,C,handler" {
		t.Errorf("order = %s, want A,B,C,handler", got)
	}
}

func TestRequestID(t *testing.T) {
	var fromContext string
	h := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fromContext = RequestIDFromContext(r.Context())
	}))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	header := rec.Header().Get("X-Request-ID")
	if header == "" || header != fromContext {
		t.Errorf("header = %q, context = %q; want the same non-empty ID", header, fromContext)
	}
}

func TestRecover(t *testing.T) {
	logs := captureLogs(t)
	h := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	assertJSONError(t, rec, http.StatusInternalServerError)
	if strings.Contains(rec.Body.String(), "boom") {
		t.Error("panic message must not leak to the client")
	}
	if !strings.Contains(logs.String(), "panic: boom") {
		t.Errorf("log = %q, want the panic to be logged", logs.String())
	}
}

func TestBasicAuth(t *testing.T) {
	captureLogs(t)
	users := map[string]string{"admin": "test-password"}

	tests := []struct {
		name       string
		setAuth    func(r *http.Request)
		wantStatus int
		wantUser   string
	}{
		{name: "valid credentials", setAuth: func(r *http.Request) { r.SetBasicAuth("admin", "test-password") }, wantStatus: http.StatusOK, wantUser: "admin"},
		{name: "missing header", setAuth: func(r *http.Request) {}, wantStatus: http.StatusUnauthorized},
		{name: "wrong password", setAuth: func(r *http.Request) { r.SetBasicAuth("admin", "nope") }, wantStatus: http.StatusUnauthorized},
		{name: "unknown user", setAuth: func(r *http.Request) { r.SetBasicAuth("bob", "test-password") }, wantStatus: http.StatusUnauthorized},
		{name: "not basic scheme", setAuth: func(r *http.Request) { r.Header.Set("Authorization", "Bearer abc") }, wantStatus: http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var gotUser string
			h := BasicAuth(users)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				gotUser, _ = UserFromContext(r.Context())
			}))
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			tt.setAuth(req)
			rec := httptest.NewRecorder()

			h.ServeHTTP(rec, req)

			if tt.wantStatus == http.StatusUnauthorized {
				assertJSONError(t, rec, http.StatusUnauthorized)
				if rec.Header().Get("WWW-Authenticate") == "" {
					t.Error("missing WWW-Authenticate header")
				}
				return
			}
			if rec.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
			if gotUser != tt.wantUser {
				t.Errorf("user in context = %q, want %q", gotUser, tt.wantUser)
			}
		})
	}
}

func TestLogging(t *testing.T) {
	logs := captureLogs(t)
	h := Chain(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}), RequestID, BasicAuth(map[string]string{"admin": "pw"}), Logging)
	req := httptest.NewRequest(http.MethodPost, "/employees?limit=5", nil)
	req.SetBasicAuth("admin", "pw")

	h.ServeHTTP(httptest.NewRecorder(), req)

	line := logs.String()
	for _, want := range []string{"POST", "/employees?limit=5", "418", "user=admin"} {
		if !strings.Contains(line, want) {
			t.Errorf("log line %q does not contain %q", line, want)
		}
	}
}

func TestTimeout(t *testing.T) {
	var deadline time.Time
	var hasDeadline bool
	h := Timeout(2 * time.Second)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		deadline, hasDeadline = r.Context().Deadline()
	}))

	h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if !hasDeadline {
		t.Fatal("request context has no deadline")
	}
	if remaining := time.Until(deadline); remaining <= 0 || remaining > 2*time.Second {
		t.Errorf("deadline is %v away, want within 2s", remaining)
	}
}
