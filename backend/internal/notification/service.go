package notification

import (
	"booking-system/pkg/mq"
	"context"
	"encoding/json"
	"log"

	"github.com/twmb/franz-go/pkg/kgo"
)

// SendSuspiciousActivityNotification queues a Kafka event so the user gets alerted about a login
// from a new device or network. Never blocks; drops and logs the event if Kafka is unavailable.
func SendSuspiciousActivityNotification(userID string, ip string, userAgent string) {
	if mq.Client == nil {
		log.Printf("kafka client unavailable, suspicious activity notification dropped for %s", userID)
		return
	}

	b, _ := json.Marshal(map[string]string{"type": "suspiciousActivity", "userID": userID, "ip": ip, "userAgent": userAgent})

	r := &kgo.Record{
		Topic: "notifications",
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
