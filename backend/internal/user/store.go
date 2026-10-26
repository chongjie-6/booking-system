package user

import (
	"booking-system/pkg/db"
	"context"

	"github.com/jackc/pgx/v5"
)

// dbCreateUser inserts a user row; the database fills in the ID and timestamps.
// Takes a context and the User; returns the saved User or the database error.
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

// dbGetUser fetches one user by ID. Takes a context and user ID; returns the User, or pgx.ErrNoRows if not found.
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

// dbUpdateUser updates a user's name and email. Takes a context, user ID and request (nil fields keep their current value
// via COALESCE); returns the updated User, or pgx.ErrNoRows if the user doesn't exist.
func dbUpdateUser(ctx context.Context, userID string, req PatchUserRequest) (User, error) {
	rows, _ := db.Pool.Query(ctx, `
	UPDATE users SET
	first_name = COALESCE(@first_name, first_name),
	last_name = COALESCE(@last_name, last_name),
	email = COALESCE(@email, email)
	WHERE user_id = @user_id
	RETURNING *
	`, pgx.NamedArgs{
		"user_id":    userID,
		"first_name": req.FirstName,
		"last_name":  req.LastName,
		"email":      req.Email,
	})

	return pgx.CollectOneRow(rows, pgx.RowToStructByName[User])
}
