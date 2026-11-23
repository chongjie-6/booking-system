// Package stripe calls the few Stripe API endpoints booking needs, over plain HTTP: payment intents, refunds, and
// webhook signature checks. Settings: STRIPE_SECRET_KEY, STRIPE_PUBLISHABLE_KEY, STRIPE_WEBHOOK_SECRET.
package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

// API is Stripe's base URL. A var so tests can point it at a fake.
var API = "https://api.stripe.com/v1"

var client = &http.Client{Timeout: 15 * time.Second}

// PaymentIntent is the part of a Stripe PaymentIntent we use. LatestCharge is only filled by GetPaymentIntent.
type PaymentIntent struct {
	ID           string            `json:"id"`
	Amount       int               `json:"amount"`
	Status       string            `json:"status"`
	ClientSecret string            `json:"client_secret"`
	Metadata     map[string]string `json:"metadata"`
	LatestCharge *struct {
		AmountRefunded int `json:"amount_refunded"`
	} `json:"latest_charge"`
}

// Refunded reports whether any of the payment has been refunded.
func (pi PaymentIntent) Refunded() bool {
	return pi.LatestCharge != nil && pi.LatestCharge.AmountRefunded > 0
}

// Error is an error answer from Stripe, e.g. 404 resource_missing.
type Error struct {
	Status  int    `json:"-"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return fmt.Sprintf("stripe %d %s: %s", e.Status, e.Code, e.Message) }

// ErrBadSignature means a webhook's Stripe-Signature header doesn't match its body, or is over 5 minutes old.
var ErrBadSignature = errors.New("bad Stripe signature")

// Enabled reports whether STRIPE_SECRET_KEY is set.
func Enabled() bool { return os.Getenv("STRIPE_SECRET_KEY") != "" }

// CreatePaymentIntent starts a card payment in Australian dollars. Takes a context, the amount in cents and metadata
// (each value at most 500 characters); returns the intent, whose ClientSecret the page pays with, or the error.
func CreatePaymentIntent(ctx context.Context, amountCents int, metadata map[string]string) (PaymentIntent, error) {
	form := url.Values{"amount": {strconv.Itoa(amountCents)}, "currency": {"aud"}, "payment_method_types[]": {"card"}}
	for k, v := range metadata {
		form.Set("metadata["+k+"]", v)
	}
	var pi PaymentIntent
	return pi, call(ctx, http.MethodPost, "/payment_intents", form, "", &pi)
}

// GetPaymentIntent fetches an intent with its latest charge, so Refunded works. Takes a context and the intent ID;
// returns the intent or the error (an *Error with Status 404 if there's no such intent).
func GetPaymentIntent(ctx context.Context, id string) (PaymentIntent, error) {
	var pi PaymentIntent
	q := url.Values{"expand[]": {"latest_charge"}}.Encode()
	return pi, call(ctx, http.MethodGet, "/payment_intents/"+url.PathEscape(id)+"?"+q, nil, "", &pi)
}

// Refund refunds a payment in full. Safe to call again for the same payment: Stripe dedupes the request for 24 hours,
// and after that an already refunded payment counts as success. Takes a context and the intent ID; returns the error.
func Refund(ctx context.Context, paymentIntentID string) error {
	err := call(ctx, http.MethodPost, "/refunds", url.Values{"payment_intent": {paymentIntentID}}, "refund-"+paymentIntentID, nil)
	if se, ok := errors.AsType[*Error](err); ok && se.Code == "charge_already_refunded" {
		return nil
	}
	return err
}

// VerifyWebhook checks a webhook body against its Stripe-Signature header ("t=<unix time>,v1=<hex HMAC-SHA256>").
// Takes the raw body, the header, the endpoint's signing secret and the current time; returns nil or ErrBadSignature.
func VerifyWebhook(payload []byte, header, secret string, now time.Time) error {
	var ts string
	var sigs []string
	for part := range strings.SplitSeq(header, ",") {
		k, v, _ := strings.Cut(part, "=")
		switch k {
		case "t":
			ts = v
		case "v1":
			sigs = append(sigs, v)
		}
	}
	t, err := strconv.ParseInt(ts, 10, 64)
	if err != nil || secret == "" || now.Sub(time.Unix(t, 0)).Abs() > 5*time.Minute {
		return ErrBadSignature
	}
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(ts + "."))
	mac.Write(payload)
	want := hex.EncodeToString(mac.Sum(nil))
	for _, s := range sigs {
		if hmac.Equal([]byte(s), []byte(want)) {
			return nil
		}
	}
	return ErrBadSignature
}

// call sends one API request and decodes the JSON answer into out (unless out is nil). A non-200 answer becomes an *Error.
func call(ctx context.Context, method, path string, form url.Values, idempotencyKey string, out any) error {
	var body io.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, API+path, body)
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "Bearer "+os.Getenv("STRIPE_SECRET_KEY"))
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	if idempotencyKey != "" {
		req.Header.Set("Idempotency-Key", idempotencyKey)
	}
	res, err := client.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode != http.StatusOK {
		var e struct{ Error Error }
		json.NewDecoder(res.Body).Decode(&e)
		e.Error.Status = res.StatusCode
		return &e.Error
	}
	if out == nil {
		return nil
	}
	return json.NewDecoder(res.Body).Decode(out)
}
