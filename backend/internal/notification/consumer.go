package notification

import (
	"booking-system/pkg/mq"
	"context"
)

// Consume sends the notification for each event on the notifications topic until ctx is cancelled.
// Run it in its own goroutine and wait for it before exiting. Takes the shutdown context; returns nothing.
func Consume(ctx context.Context) {
	mq.Consume(ctx, "notification", mq.NotificationsTopic, map[string]mq.Handler{
		suspiciousActivity: sendSuspiciousActivityEmail,
		passwordReset:      sendPasswordResetEmail,
	})
}
