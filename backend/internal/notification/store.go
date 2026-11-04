package notification

import (
	"booking-system/pkg/db"
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// dbInsertResetToken saves a password reset token's hash. Takes a context, the hash, user ID and expiry; returns the database error, if any.
func dbInsertResetToken(ctx context.Context, tokenHash string, userID string, expiresAt time.Time) error {
	_, err := db.Pool.Exec(ctx, `
	INSERT INTO password_reset_tokens (token_hash, user_id, expires_at)
	VALUES (@token_hash, @user_id, @expires_at)
	`, pgx.NamedArgs{"token_hash": tokenHash, "user_id": userID, "expires_at": expiresAt})
	return err
}
