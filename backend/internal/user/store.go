package user

import (
	"booking-system/pkg/db"
	"context"

	"github.com/jackc/pgx/v5"
)

func dbCreateUser(ctx context.Context, user User) (User, error) {
	rows, _ := db.Pool.Query(ctx, `
	INSERT INTO
	users(user_role, hashed_password, first_name, last_name, email)
	VALUES (@user_role, @hashed_password, @first_name, @last_name, @email)
	RETURNING *
	`, pgx.NamedArgs{
		"user_role":       user.UserRole,
		"hashed_password": user.HashedPassword,
		"first_name":      user.FirstName,
		"last_name":       user.LastName,
		"email":           user.Email,
	})
	return pgx.CollectOneRow(rows, pgx.RowToStructByName[User])
}

func dbGetUser(ctx context.Context, userID string) (User, error) {
	rows, _ := db.Pool.Query(ctx, `
	SELECT 
	user_id, user_role, hashed_password, first_name, last_name, email, created_at, updated_at
	FROM users
	WHERE user_id = @user_id
	`, pgx.NamedArgs{
		"user_id": userID,
	})

	return pgx.CollectOneRow(rows, pgx.RowToStructByName[User])
}

// Returns a nil hash when the user has no password set, and pgx.ErrNoRows when the user doesn't exist.
func dbGetPasswordHash(ctx context.Context, userID string) (*string, error) {
	var hash *string
	err := db.Pool.QueryRow(ctx, `
	SELECT 
	hashed_password 
	FROM users 
	WHERE user_id = @user_id
	`, pgx.NamedArgs{
		"user_id": userID,
	}).Scan(&hash)
	return hash, err
}

// A nil field is sent as NULL, so COALESCE keeps the current value.
func dbUpdateUser(ctx context.Context, userID string, req PatchUserRequest, hashedPassword *string) (User, error) {
	rows, _ := db.Pool.Query(ctx, `
	UPDATE users SET
	first_name = COALESCE(@first_name, first_name),
	last_name = COALESCE(@last_name, last_name),
	email = COALESCE(@email, email),
	hashed_password = COALESCE(@hashed_password, hashed_password)
	WHERE user_id = @user_id
	RETURNING *
	`, pgx.NamedArgs{
		"user_id":         userID,
		"first_name":      req.FirstName,
		"last_name":       req.LastName,
		"email":           req.Email,
		"hashed_password": hashedPassword,
	})

	return pgx.CollectOneRow(rows, pgx.RowToStructByName[User])
}

// Satisfied by both db.Pool and pgx.Tx.
type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
}

// Revokes the token if it's still valid and returns its row; pgx.ErrNoRows means it was unknown, expired or already used.
// The UPDATE locks the row, so a concurrent request with the same token waits, then matches nothing.
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
