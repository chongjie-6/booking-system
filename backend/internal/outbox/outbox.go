package outbox

import (
	"booking-system/pkg/db"
	"booking-system/pkg/mq"
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/twmb/franz-go/pkg/kgo"
)

const batchSize = 100

// Relay sends unsent outbox rows to Kafka, oldest first, until ctx is cancelled, and hourly deletes rows sent over a week ago.
// Run it in its own goroutine and wait for it before closing the producer. Returns nothing, logging failures.
func Relay(ctx context.Context) {
	if mq.Client == nil {
		log.Print("outbox relay: no kafka client, events stay queued until restart")
		return
	}
	tick := time.NewTicker(time.Second)
	defer tick.Stop()
	prune := time.NewTicker(time.Hour)
	defer prune.Stop()
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
		case <-prune.C:
			if err := dbPrunePublished(); err != nil {
				log.Printf("outbox prune: %v", err)
			}
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
