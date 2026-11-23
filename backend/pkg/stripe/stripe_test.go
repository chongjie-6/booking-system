package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"
)

// TestCalls checks requests carry the key, form fields and idempotency key, the charge is expanded,
// Stripe errors decode, and refunding an already refunded payment counts as success.
func TestCalls(t *testing.T) {
	t.Setenv("STRIPE_SECRET_KEY", "sk_test")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("Authorization") != "Bearer sk_test" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		r.ParseForm()
		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/payment_intents":
			if r.Form.Get("amount") != "3300" || r.Form.Get("currency") != "aud" || r.Form.Get("metadata[event_id]") != "e1" {
				w.WriteHeader(http.StatusBadRequest)
				return
			}
			w.Write([]byte(`{"id":"pi_1","amount":3300,"status":"requires_payment_method","client_secret":"pi_1_secret","latest_charge":null}`))
		case r.Method == http.MethodGet && r.URL.Path == "/payment_intents/pi_1" && r.URL.Query().Get("expand[]") == "latest_charge":
			w.Write([]byte(`{"id":"pi_1","amount":3300,"status":"succeeded","metadata":{"event_id":"e1"},"latest_charge":{"amount_refunded":3300}}`))
		case r.Method == http.MethodPost && r.URL.Path == "/refunds" && r.Header.Get("Idempotency-Key") == "refund-"+r.Form.Get("payment_intent"):
			w.WriteHeader(http.StatusBadRequest)
			w.Write([]byte(`{"error":{"code":"charge_already_refunded","message":"already refunded"}}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":{"code":"resource_missing","message":"no such thing"}}`))
		}
	}))
	defer srv.Close()
	API = srv.URL
	ctx := context.Background()

	pi, err := CreatePaymentIntent(ctx, 3300, map[string]string{"event_id": "e1"})
	if err != nil || pi.ClientSecret != "pi_1_secret" || pi.Refunded() {
		t.Fatalf("CreatePaymentIntent = %+v, %v", pi, err)
	}
	pi, err = GetPaymentIntent(ctx, "pi_1")
	if err != nil || pi.Status != "succeeded" || pi.Metadata["event_id"] != "e1" || !pi.Refunded() {
		t.Fatalf("GetPaymentIntent = %+v, %v", pi, err)
	}
	if err := Refund(ctx, "pi_1"); err != nil {
		t.Fatalf("Refund of a refunded payment = %v, want nil", err)
	}
	_, err = GetPaymentIntent(ctx, "pi_missing")
	if se, ok := err.(*Error); !ok || se.Status != http.StatusNotFound || se.Code != "resource_missing" {
		t.Fatalf("GetPaymentIntent of a missing intent = %v, want a 404 *Error", err)
	}
}

// TestVerifyWebhook checks a correctly signed body passes, and a wrong secret, edited body, stale or missing timestamp fail.
func TestVerifyWebhook(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	body := []byte(`{"type":"payment_intent.succeeded"}`)
	sign := func(secret string, at time.Time, payload []byte) string {
		ts := strconv.FormatInt(at.Unix(), 10)
		mac := hmac.New(sha256.New, []byte(secret))
		mac.Write([]byte(ts + "."))
		mac.Write(payload)
		return "t=" + ts + ",v1=" + hex.EncodeToString(mac.Sum(nil))
	}

	if err := VerifyWebhook(body, sign("whsec", now, body)+",v0=old", "whsec", now.Add(time.Minute)); err != nil {
		t.Fatalf("good signature = %v", err)
	}
	bad := map[string]string{
		"wrong secret": sign("other", now, body),
		"edited body":  sign("whsec", now, []byte(`{}`)),
		"stale":        sign("whsec", now.Add(-6*time.Minute), body),
		"no timestamp": "v1=abc",
	}
	for name, header := range bad {
		if err := VerifyWebhook(body, header, "whsec", now); err != ErrBadSignature {
			t.Errorf("%s = %v, want ErrBadSignature", name, err)
		}
	}
	if err := VerifyWebhook(body, sign("", now, body), "", now); err != ErrBadSignature {
		t.Errorf("empty secret = %v, want ErrBadSignature", err)
	}
}
