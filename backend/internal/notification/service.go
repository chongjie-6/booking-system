package notification

import (
	"booking-system/internal/outbox"
	"booking-system/internal/user"
	"booking-system/pkg/auth"
	"booking-system/pkg/db"
	"booking-system/pkg/email"
	"booking-system/pkg/mq"
	"context"
	"crypto/rand"
	"embed"
	"encoding/json"
	"html/template"
	"log"
	"os"
	"strings"
	"time"
)

const (
	suspiciousActivity = "suspiciousActivity"
	passwordReset      = "passwordReset"
)

// resetTTL is how long a password reset link works, counted from when the email is sent.
const resetTTL = time.Hour

// templateFS holds the email bodies, embedded in the binary so the templates folder isn't needed at runtime.
//
//go:embed templates/*.html
var templateFS embed.FS

// templates are the parsed email bodies, looked up by file name. Parsed once at startup; a broken template panics here, not mid-send.
// html/template escapes every value, so attacker-controlled input like the user agent can't inject markup or links into the email.
var templates = template.Must(template.ParseFS(templateFS, "templates/*.html"))

// SendSuspiciousActivityNotification saves an alert event to the outbox for Debezium to send to Kafka.
// Logs and drops it if the insert fails, so a database error never fails the login.
func SendSuspiciousActivityNotification(ctx context.Context, userID string, ip string, userAgent string) {
	e := event{Type: suspiciousActivity, UserID: userID, IP: ip, UserAgent: userAgent}
	if err := outbox.Enqueue(ctx, db.Pool, mq.NotificationsTopic, userID, e); err != nil {
		log.Printf("suspicious activity notification for %s: %v", userID, err)
	}
}

// SendPasswordResetNotification saves a password reset event to the outbox for Debezium to send to Kafka.
// Takes a context and user ID; returns the insert error, if any.
func SendPasswordResetNotification(ctx context.Context, userID string) error {
	return outbox.Enqueue(ctx, db.Pool, mq.NotificationsTopic, userID, event{Type: passwordReset, UserID: userID})
}

// sendPasswordResetEmail makes a reset token and emails the user a link to the frontend with it.
// The token is made here rather than when the event is queued, so the raw token never sits in the outbox or Kafka.
// A retry makes another token; any of them works until one is used. Takes a context and the event's JSON;
// returns the decode, user lookup, insert, template or send error.
func sendPasswordResetEmail(ctx context.Context, b []byte) error {
	var e event
	if err := json.Unmarshal(b, &e); err != nil {
		return err
	}
	u, err := user.GetUserService(ctx, e.UserID)
	if err != nil {
		return err
	}
	token := rand.Text()
	if err := dbInsertResetToken(ctx, auth.HashRefreshToken(token), u.UserID, time.Now().Add(resetTTL)); err != nil {
		return err
	}
	// ALLOWED_ORIGINS is the one frontend origin. The token goes in the fragment, which browsers never send to a server.
	data := struct{ FirstName, Link string }{u.FirstName, os.Getenv("ALLOWED_ORIGINS") + "/#reset=" + token}
	var body strings.Builder
	if err := templates.ExecuteTemplate(&body, "password_reset.html", data); err != nil {
		return err
	}
	return email.Send(ctx, u.Email, "Reset your password", body.String())
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
