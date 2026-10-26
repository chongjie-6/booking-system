package auth

import (
	"errors"
	"testing"
)

// Login relies on ErrPasswordMismatch to tell a wrong password (401) from a corrupt hash (500).
func TestVerifyPassword(t *testing.T) {
	hash := HashPassword("correct horse")

	if err := VerifyPassword("correct horse", hash); err != nil {
		t.Fatalf("right password: got %v, want nil", err)
	}
	if err := VerifyPassword("wrong horse", hash); !errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("wrong password: got %v, want ErrPasswordMismatch", err)
	}
	if err := VerifyPassword("correct horse", "not-a-hash"); err == nil || errors.Is(err, ErrPasswordMismatch) {
		t.Fatalf("corrupt hash: got %v, want a different error", err)
	}
}
