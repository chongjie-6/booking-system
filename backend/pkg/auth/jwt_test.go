package auth

import (
	"context"
	"errors"
	"testing"
	"time"
)

// Login relies on ErrPasswordMismatch to tell a wrong password (401) from a corrupt hash (500).
func TestVerifyPassword(t *testing.T) {
	ctx := context.Background()
	hash, err := HashPassword(ctx, "correct horse")
	if err != nil {
		t.Fatal(err)
	}

	if err := VerifyPassword(ctx, "correct horse", hash); err != nil {
		t.Fatalf("right password: got %v, want nil", err)
	}
	if err := VerifyPassword(ctx, "wrong horse", hash); !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("wrong password: got %v, want ErrPasswordMismatch", err)
	}
	if err := VerifyPassword(ctx, "correct horse", "not-a-hash"); err == nil || errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("corrupt hash: got %v, want a different error", err)
	}
}

// With every argon2 slot taken, a hash waits, and gives up once its request's context ends.
func TestHashWaitsForSlot(t *testing.T) {
	for range cap(argonSlots) {
		argonSlots <- struct{}{}
	}
	defer func() {
		for range cap(argonSlots) {
			<-argonSlots
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if _, err := HashPassword(ctx, "correct horse"); !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("got %v, want context.DeadlineExceeded", err)
	}
}
