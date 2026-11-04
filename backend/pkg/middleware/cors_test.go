package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestCORS checks each kind of request gets the status and Allow-Origin header the browser needs.
// Takes the test runner; fails the test on any mismatch.
func TestCORS(t *testing.T) {
	t.Setenv("ALLOWED_ORIGINS", "http://localhost:5173")
	h := CORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusTeapot) }))

	cases := []struct {
		method, origin string
		status         int
		allowOrigin    string
	}{
		{"GET", "", http.StatusTeapot, ""},                                                  // no Origin: passes through
		{"GET", "http://example.com", http.StatusTeapot, ""},                                // same host as the request
		{"GET", "http://evil.test", http.StatusForbidden, ""},                               // other origin
		{"POST", "http://localhost:5173", http.StatusTeapot, "http://localhost:5173"},       // allowed origin
		{"OPTIONS", "http://localhost:5173", http.StatusNoContent, "http://localhost:5173"}, // preflight, handler not run
	}
	for _, c := range cases {
		r := httptest.NewRequest(c.method, "http://example.com/auth/login", nil)
		if c.origin != "" {
			r.Header.Set("Origin", c.origin)
		}
		w := httptest.NewRecorder()
		h.ServeHTTP(w, r)
		if w.Code != c.status || w.Header().Get("Access-Control-Allow-Origin") != c.allowOrigin {
			t.Errorf("%s from %q: got %d %q, want %d %q", c.method, c.origin,
				w.Code, w.Header().Get("Access-Control-Allow-Origin"), c.status, c.allowOrigin)
		}
	}
}
