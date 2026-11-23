package middleware

import (
	"net/http/httptest"
	"testing"
)

// A forged X-Real-IP must not dodge the rate limits unless a proxy that overwrites it is the only way in.
func TestClientIP(t *testing.T) {
	r := httptest.NewRequest("GET", "/", nil)
	r.RemoteAddr = "10.0.0.2:5000"
	r.Header.Set("X-Real-IP", "203.0.113.7")

	if got := ClientIP(r); got != "10.0.0.2" {
		t.Errorf("without TRUST_PROXY = %q, want the peer", got)
	}
	t.Setenv("TRUST_PROXY", "true")
	if got := ClientIP(r); got != "203.0.113.7" {
		t.Errorf("with TRUST_PROXY = %q, want X-Real-IP", got)
	}
	r.Header.Del("X-Real-IP")
	if got := ClientIP(r); got != "10.0.0.2" {
		t.Errorf("with TRUST_PROXY and no header = %q, want the peer", got)
	}
}
