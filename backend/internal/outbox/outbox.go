// Package outbox saves events in the same transaction as the write that caused them. Debezium reads new rows from the
// Postgres WAL and sends them to Kafka (see debezium/outbox.json); this package only writes rows and prunes old ones.
package outbox

import (
	"booking-system/pkg/rdb"
	"context"
	"log"
	"time"
)

// Prune hourly deletes rows created over a week ago until ctx is cancelled. Debezium sent them long before, and its outbox
// router ignores deletes, so pruning sends nothing. The Redis lock means one instance prunes per hour;
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
		if err := dbPrune(); err != nil {
			log.Printf("outbox prune: %v", err)
		}
	}
}
