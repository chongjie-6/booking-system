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
