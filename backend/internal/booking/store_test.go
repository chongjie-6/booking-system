package booking

import (
	"booking-system/pkg/rdb"
	"context"
	"errors"
	"slices"
	"testing"
)

// TestHolds checks seat holds: all-or-nothing between users, refreshable by the holder, released only by the holder,
// and skipped when Redis is down. The Redis-backed part runs only when Redis answers on localhost:6379, as in docker-compose.
func TestHolds(t *testing.T) {
	ctx := context.Background()

	t.Setenv("REDIS_ADDRESS", "127.0.0.1:1") // nothing listens here
	rdb.Connect(ctx)
	if err := hold(ctx, "e", "alice", []string{"a"}); err != nil {
		t.Fatalf("hold with Redis down = %v, want nil", err)
	}
	if h := Holders(ctx, "e", []string{"a"}); h != nil {
		t.Fatalf("Holders with Redis down = %v, want nil", h)
	}

	t.Setenv("REDIS_ADDRESS", "localhost:6379")
	rdb.Connect(ctx)
	if rdb.Client.Ping(ctx).Err() != nil {
		t.Skip("Redis unreachable; skipping the held-seat checks")
	}
	event := "test-" + t.Name()
	t.Cleanup(func() { rdb.Client.Del(ctx, holdKeys(event, []string{"a", "b", "c"})...) })

	if err := hold(ctx, event, "alice", []string{"a", "b"}); err != nil {
		t.Fatalf("alice hold: %v", err)
	}
	if err := hold(ctx, event, "alice", []string{"a", "b"}); err != nil {
		t.Fatalf("alice re-hold: %v", err)
	}
	if err := hold(ctx, event, "bob", []string{"b", "c"}); !errors.Is(err, errSeatsTaken) {
		t.Fatalf("bob hold over alice = %v, want errSeatsTaken", err)
	}
	if h := Holders(ctx, event, []string{"a", "b", "c"}); !slices.Equal(h, []string{"alice", "alice", ""}) {
		t.Fatalf("Holders = %q, want alice, alice, none (bob's clash must hold nothing)", h)
	}
	release(ctx, event, "bob", []string{"a"})
	release(ctx, event, "alice", []string{"b"})
	if h := Holders(ctx, event, []string{"a", "b"}); !slices.Equal(h, []string{"alice", ""}) {
		t.Fatalf("after releases Holders = %q, want alice, none", h)
	}
}
