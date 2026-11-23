package middleware

import (
	"booking-system/pkg/auth"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/x509"
	"encoding/pem"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestAdmin checks only an admin's token reaches the handler: no token is 401, a user's token is 403.
func TestAdmin(t *testing.T) {
	key, _ := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	der, _ := x509.MarshalECPrivateKey(key)
	t.Setenv("JWT_PRIVATE_KEY", string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: der})))

	h := Admin(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusTeapot) })
	status := func(role string) int {
		r := httptest.NewRequest(http.MethodGet, "/admin/movies", nil)
		if role != "" {
			token, err := auth.NewAccessToken("u1", role)
			if err != nil {
				t.Fatal(err)
			}
			r.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		}
		w := httptest.NewRecorder()
		h(w, r)
		return w.Code
	}

	for role, want := range map[string]int{"": http.StatusUnauthorized, "user": http.StatusForbidden, "admin": http.StatusTeapot} {
		if got := status(role); got != want {
			t.Errorf("role %q = %d, want %d", role, got, want)
		}
	}
}
