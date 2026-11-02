package outbox

import (
	"booking-system/pkg/db"
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

type outboxRow struct {
	OutboxID string
	Topic    string
	Key      string
	Payload  []byte
}

// Enqueue saves an event for the relay to send to topic.
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

// dbLockUnsent locks up to limit unsent rows, oldest first, skipping rows another relay already holds.
func dbLockUnsent(ctx context.Context, tx pgx.Tx, limit int) ([]outboxRow, error) {
	rows, _ := tx.Query(ctx, `
	SELECT outbox_id, topic, key, payload FROM outbox
	WHERE published_at IS NULL
	ORDER BY outbox_id
	LIMIT @limit
	FOR UPDATE SKIP LOCKED`, pgx.NamedArgs{"limit": limit})
	return pgx.CollectRows(rows, pgx.RowToStructByPos[outboxRow])
}

// dbMarkPublished stamps published_at on the given rows.
func dbMarkPublished(ctx context.Context, tx pgx.Tx, ids []string) error {
	_, err := tx.Exec(ctx, `UPDATE outbox SET published_at = now() WHERE outbox_id = ANY(@ids)`, pgx.NamedArgs{"ids": ids})
	return err
}

// dbPrunePublished deletes rows sent more than a week ago, so the table doesn't grow forever.
func dbPrunePublished() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_, err := db.Pool.Exec(ctx, `DELETE FROM outbox WHERE published_at < now() - interval '7 days'`)
	return err
}
