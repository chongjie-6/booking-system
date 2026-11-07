package outbox

import (
	"booking-system/pkg/db"
	"booking-system/pkg/mq"
	"booking-system/pkg/rdb"
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/twmb/franz-go/pkg/kgo"
)

const batchSize = 100

// Relay sends unsent outbox rows to Kafka, oldest first, until ctx is cancelled.
// Run it in its own goroutine and wait for it before closing the producer. Returns nothing, logging failures.
func Relay(ctx context.Context) {
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	failing := false
	for {
		n, err := relayBatch()
		// Log only when sending starts failing or recovers, not every tick of an outage.
		if err != nil && !failing {
			log.Printf("outbox relay: %v", err)
		} else if err == nil && failing {
			log.Print("outbox relay: recovered")
		}
		failing = err != nil
		if n == batchSize && ctx.Err() == nil {
			continue // a full batch means more are waiting
		}
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
	}
}

// Prune hourly deletes rows sent over a week ago until ctx is cancelled. The Redis lock means one instance prunes per hour;
// if Redis is down every instance prunes, which is safe since the delete is idempotent.
func Prune(ctx context.Context) {
	tick := time.NewTicker(time.Hour)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
		}
		// The TTL is shorter than the interval so it has expired by the next tick, and is never released, so other instances skip this hour.
		if ok, err := rdb.AcquireLock(ctx, "outbox:prune", 50*time.Minute); err == nil && !ok {
			continue
		}
		if err := dbPrunePublished(); err != nil {
			log.Printf("outbox prune: %v", err)
		}
	}
}

// relayBatch sends up to batchSize rows in order and marks the ones Kafka accepted. It stops at the first failed send,
// so with a single relay one key's events never go out of order; the rest retry next tick.
// Returns how many were sent, and the send or database error that stopped it, if any.
func relayBatch() (int, error) {
	// Not the shutdown context: a started batch should finish, so the rows it sent get marked.
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	// Sends get less time than the transaction, so a slow broker can't leave too little to mark the sent rows and commit.
	sendCtx, cancelSend := context.WithTimeout(ctx, 8*time.Second)
	defer cancelSend()

	var sent []string
	var sendErr error
	err := pgx.BeginFunc(ctx, db.Pool, func(tx pgx.Tx) error {
		rows, err := dbLockUnsent(ctx, tx, batchSize)
		if err != nil {
			return err
		}
		for _, r := range rows {
			rec := &kgo.Record{
				Topic:   r.Topic,
				Key:     []byte(r.Key),
				Value:   r.Payload,
				Headers: []kgo.RecordHeader{{Key: "event_id", Value: []byte(r.OutboxID)}},
			}
			if sendErr = mq.Produce(sendCtx, rec); sendErr != nil {
				break
			}
			sent = append(sent, r.OutboxID)
		}
		return dbMarkPublished(ctx, tx, sent)
	})
	if err != nil {
		return 0, err
	}
	return len(sent), sendErr
}
