package user

import (
	"booking-system/pkg/auth"
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

var ErrEmailTaken = errors.New("email already registered")

// CreateUserService hashes the password and stores a new user with the "user" role.
// Takes a context and CreateUserRequest; returns the created User, or ErrEmailTaken if the email is already registered.
func CreateUserService(ctx context.Context, req CreateUserRequest) (User, error) {
	hashed := auth.HashPassword(req.Password)

	u, err := dbCreateUser(ctx, User{
		UserRole:       "user",
		HashedPassword: &hashed,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Email:          req.Email,
	})

	// 23505 = unique violation; the index on lower(email) catches duplicates
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_key" {
		return User{}, ErrEmailTaken
	}
	return u, err
}

// GetUserService looks up a user by ID. Takes a context and user ID; returns the User, or pgx.ErrNoRows if not found.
func GetUserService(ctx context.Context, userID string) (User, error) {
	return dbGetUser(ctx, userID)
}

// UpdateUserService applies a PatchUserRequest. Passwords are changed through /auth/password instead.
// Takes a context, user ID and request; returns the updated User, or ErrEmailTaken or pgx.ErrNoRows.
func UpdateUserService(ctx context.Context, userID string, req PatchUserRequest) (User, error) {
	u, err := dbUpdateUser(ctx, userID, req)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_key" {
		return User{}, ErrEmailTaken
	}

	return u, err
}
