// Package email sends emails to users
package email

import (
	"context"
	"errors"
	"log"
	"os"

	"github.com/resend/resend-go/v4"
)

var (
	client *resend.Client // shared Resend client, set by Connect
	from   string         // its domain must be verified in Resend
)

// ErrNotConfigured is returned by Send when Connect didn't set up a client.
var ErrNotConfigured = errors.New("email not configured")

// Connect creates the shared Resend client from RESEND_API_KEY and reads the sender from EMAIL_FROM.
// Takes no input and returns nothing; logs and leaves the client nil if either variable is unset, so Send returns ErrNotConfigured.
func Connect() {
	key, f := os.Getenv("RESEND_API_KEY"), os.Getenv("EMAIL_FROM")
	if key == "" || f == "" {
		log.Print("email disabled: RESEND_API_KEY and EMAIL_FROM must be set")
		return
	}
	client, from = resend.NewClient(key), f
}

// Send emails an HTML body to one address. Call Connect first.
// Takes a context (its deadline cancels the request), the recipient, a subject and the HTML body, already escaped
// (render it with html/template); returns ErrNotConfigured if there's no client, otherwise the Resend error, if any.
func Send(ctx context.Context, to string, subject string, html string) error {
	if client == nil {
		return ErrNotConfigured
	}
	params := &resend.SendEmailRequest{
		From:    from,
		To:      []string{to},
		Subject: subject,
		Html:    html,
	}

	_, err := client.Emails.SendWithContext(ctx, params)
	return err
}
