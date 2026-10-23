package user

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"
)

// Hashes the password and stores a new user with the "user" role.
func CreateUserService(ctx context.Context, req CreateUserRequest) (User, error) {
	// hash the password max=72 counts characters, bcrypt counts bytes, so emoji/accents can still overflow
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)

	if err != nil {
		return User{}, err
	}

	hashed := string(hash)

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

func GetUserService(ctx context.Context, userID string) (User, error) {
	return dbGetUser(ctx, userID)
}

// Hashes a new password if one was sent, then applies the changes.
func UpdateUserService(ctx context.Context, userID string, req PatchUserRequest) (User, error) {
	var hashed *string
	if req.Password != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return User{}, err
		}
		h := string(hash)
		hashed = &h
	}

	return dbUpdateUser(ctx, userID, req, hashed)
}
