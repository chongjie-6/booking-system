package notification

import (
	"booking-system/internal/user"
	"booking-system/pkg/email"
	"booking-system/pkg/mq"
	"context"
	"embed"
	"encoding/json"
	"html/template"
	"log"
	"strings"

	"github.com/twmb/franz-go/pkg/kgo"
)

const suspiciousActivity = "suspiciousActivity"

// templateFS holds the email bodies, embedded in the binary so the templates folder isn't needed at runtime.
//
//go:embed templates/*.html
var templateFS embed.FS

// templates are the parsed email bodies, looked up by file name. Parsed once at startup; a broken template panics here, not mid-send.
// html/template escapes every value, so attacker-controlled input like the user agent can't inject markup or links into the email.
var templates = template.Must(template.ParseFS(templateFS, "templates/*.html"))

// SendSuspiciousActivityNotification queues a Kafka event so the user gets alerted about a login
// from a new device or network. Never blocks; drops and logs the event if Kafka is unavailable.
func SendSuspiciousActivityNotification(userID string, ip string, userAgent string) {
	if mq.Client == nil {
		log.Printf("kafka client unavailable, suspicious activity notification dropped for %s", userID)
		return
	}

	b, _ := json.Marshal(event{Type: suspiciousActivity, UserID: userID, IP: ip, UserAgent: userAgent})

	r := &kgo.Record{
		Topic: mq.NotificationsTopic,
		Key:   []byte(userID),
		Value: b,
	}

	// context.Background, not the request's: franz-go drops a record whose context is cancelled, and the request's ends on response.
	// TryProduce, not Produce: it fails instead of blocking the login when the send buffer is full.
	mq.Client.TryProduce(context.Background(), r, func(_ *kgo.Record, err error) {
		if err != nil {
			log.Printf("suspicious activity notification: %v", err)
		}
	})
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
