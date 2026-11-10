package outbox

import (
	"booking-system/pkg/db"
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// Enqueue saves an event for Debezium to send to topic, with key as the Kafka message key.
// Pass the caller's transaction so it commits with the caller's write, or db.Pool when there isn't one.
func Enqueue(ctx context.Context, q db.Querier, topic, key string, event any) error {
	_, err := q.Exec(ctx, `
	INSERT INTO outbox (topic, key, payload)
	VALUES (@topic, @key, @payload)
	`, pgx.NamedArgs{
		"topic":   topic,
		"key":     key,
		"payload": event,
	})
	return err
}

// dbPrune deletes rows created more than a week ago, so the table doesn't grow forever.
func dbPrune() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := db.Pool.Exec(ctx, `DELETE FROM outbox WHERE created_at < now() - interval '7 days'`)
	return err
}
