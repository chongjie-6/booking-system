package auth

import (
	"booking-system/internal/user"
	"booking-system/pkg/db"
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Satisfied by both db.Pool and pgx.Tx.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// dbSetPasswordAndRevokeTokens saves a new password hash and revokes every live refresh token the user has.
// Takes a context, a Pool or Tx, the user ID and new hash; returns the database error, if any.
func dbSetPasswordAndRevokeTokens(ctx context.Context, q querier, userID string, hash string) error {
	args := pgx.NamedArgs{"user_id": userID, "hash": hash}
	if _, err := q.Exec(ctx, `UPDATE users SET hashed_password = @hash WHERE user_id = @user_id`, args); err != nil {
		return err
	}
	_, err := q.Exec(ctx, `UPDATE refresh_tokens SET is_revoked = true WHERE user_id = @user_id AND is_revoked = false`, args)
	return err
}

// dbRevokeRefreshToken revokes a still-valid refresh token. Takes a context, a Pool or Tx, and the token hash; returns the old row,
// or pgx.ErrNoRows if it was unknown, expired or already used. The UPDATE locks the row, so a concurrent reuse matches nothing.
func dbRevokeRefreshToken(ctx context.Context, q querier, tokenHash string) (RefreshTokenRow, error) {
	rows, _ := q.Query(ctx, `
	WITH rt AS (
		UPDATE refresh_tokens
		SET is_revoked = true
		WHERE token_hash = @token_hash AND is_revoked = false AND expires_at > now()
		RETURNING token_hash, user_id, expires_at, is_revoked, ip_address, user_agent
	)
	SELECT
	rt.token_hash, rt.user_id, u.user_role, rt.expires_at, rt.is_revoked, rt.ip_address, rt.user_agent
	FROM rt
	JOIN users u ON u.user_id = rt.user_id
	`, pgx.NamedArgs{
		"token_hash": tokenHash,
	})
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[RefreshTokenRow])
}

// dbInsertRefreshToken saves a new refresh token row. Takes a context, a Pool or Tx, and the row;
// returns the saved row (with the user's role) or the database error.
func dbInsertRefreshToken(ctx context.Context, q querier, refreshToken RefreshTokenRow) (RefreshTokenRow, error) {
	rows, _ := q.Query(ctx, `
	WITH rt AS (
		INSERT INTO
		refresh_tokens(user_id, token_hash, expires_at, is_revoked, ip_address, user_agent)
		VALUES (@user_id, @token_hash, @expires_at, @is_revoked, @ip_address, @user_agent)
		RETURNING token_hash, user_id, expires_at, is_revoked, ip_address, user_agent
	)
	SELECT
	rt.token_hash, rt.user_id, u.user_role, rt.expires_at, rt.is_revoked, rt.ip_address, rt.user_agent
	FROM rt
	JOIN users u ON u.user_id = rt.user_id
	`, pgx.NamedArgs{
		"user_id":    refreshToken.UserID,
		"token_hash": refreshToken.TokenHash,
		"expires_at": refreshToken.ExpiresAt,
		"is_revoked": refreshToken.IsRevoked,
		"ip_address": refreshToken.IPAddress,
		"user_agent": refreshToken.UserAgent,
	})
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[RefreshTokenRow])
}

// dbGetUserPasswordHashFromEmail fetches a user's ID, role and password hash by email, ignoring case. Takes a context and email;
// returns a User with only those fields set, a nil hash if no password is set, or pgx.ErrNoRows if no user has that email.
func dbGetUserPasswordHashFromEmail(ctx context.Context, email string) (user.User, error) {
	rows, _ := db.Pool.Query(ctx, `
	SELECT
	user_id, user_role, hashed_password
	FROM users
	WHERE lower(email) = lower(@email)
	`, pgx.NamedArgs{
		"email": email,
	})
	return pgx.CollectOneRow(rows, pgx.RowToStructByNameLax[user.User])
}
