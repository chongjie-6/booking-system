package notification

import (
	"booking-system/internal/outbox"
	"booking-system/pkg/db"
	"booking-system/pkg/mq"
	"context"
	"log"
)

const (
	suspiciousActivity = "suspiciousActivity"
	passwordReset      = "passwordReset"
)

// SendSuspiciousActivityNotification saves an alert event to the outbox for the relay to send.
// Logs and drops it if the insert fails, so a database error never fails the login.
func SendSuspiciousActivityNotification(ctx context.Context, userID string, ip string, userAgent string) {
	e := event{Type: suspiciousActivity, UserID: userID, IP: ip, UserAgent: userAgent}
	if err := outbox.Enqueue(ctx, db.Pool, mq.NotificationsTopic, userID, e); err != nil {
		log.Printf("suspicious activity notification for %s: %v", userID, err)
	}
}

// SendPasswordResetNotification saves a password reset event to the outbox for the relay to send.
// Takes a context and user ID; returns the insert error, if any.
func SendPasswordResetNotification(ctx context.Context, userID string) error {
	return outbox.Enqueue(ctx, db.Pool, mq.NotificationsTopic, userID, event{Type: passwordReset, UserID: userID})
}
