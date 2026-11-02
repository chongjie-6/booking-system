package notification

import (
	"booking-system/internal/outbox"
	"booking-system/internal/user"
	"booking-system/pkg/db"
	"booking-system/pkg/email"
	"booking-system/pkg/mq"
	"context"
	"embed"
	"encoding/json"
	"html/template"
	"log"
	"strings"
)

const suspiciousActivity = "suspiciousActivity"

// templateFS holds the email bodies, embedded in the binary so the templates folder isn't needed at runtime.
//
//go:embed templates/*.html
var templateFS embed.FS

// templates are the parsed email bodies, looked up by file name. Parsed once at startup; a broken template panics here, not mid-send.
// html/template escapes every value, so attacker-controlled input like the user agent can't inject markup or links into the email.
var templates = template.Must(template.ParseFS(templateFS, "templates/*.html"))

// SendSuspiciousActivityNotification saves an alert event to the outbox for the relay to send.
// Logs and drops it if the insert fails, so a database error never fails the login.
func SendSuspiciousActivityNotification(ctx context.Context, userID string, ip string, userAgent string) {
	e := event{Type: suspiciousActivity, UserID: userID, IP: ip, UserAgent: userAgent}
	if err := outbox.Enqueue(ctx, db.Pool, mq.NotificationsTopic, userID, e); err != nil {
		log.Printf("suspicious activity notification for %s: %v", userID, err)
	}
}

// Consume sends the notification for each event on the notifications topic until ctx is cancelled.
// Run it in its own goroutine and wait for it before exiting. Takes the shutdown context; returns nothing.
func Consume(ctx context.Context) {
	mq.Consume(ctx, "notification", mq.NotificationsTopic, map[string]mq.Handler{
		suspiciousActivity: sendSuspiciousActivityEmail,
	})
}

// sendSuspiciousActivityEmail emails the user about a login from a new device or network.
// Takes a context and the event's JSON; returns the decode, user lookup or send error.
func sendSuspiciousActivityEmail(ctx context.Context, b []byte) error {
	var e event
	if err := json.Unmarshal(b, &e); err != nil {
		return err
	}
	u, err := user.GetUserService(ctx, e.UserID)
	if err != nil {
		return err
	}
	var body strings.Builder
	data := struct{ FirstName, IP, Device string }{u.FirstName, e.IP, e.UserAgent}
	if err := templates.ExecuteTemplate(&body, "suspicious_activity.html", data); err != nil {
		return err
	}
	return email.Send(ctx, u.Email, "New login to your account", body.String())
}
