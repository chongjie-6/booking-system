package user

import (
	"booking-system/pkg/auth"
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrEmailTaken    = errors.New("email already registered")
	ErrWrongPassword = errors.New("current password is incorrect")
	ErrBadToken      = errors.New("jwt could not be issued")
)

func AttachUserCookies(ctx *gin.Context, userID string, userRole string) error {
	// issue new jwt
	jwt, err := auth.NewJWT(userID, userRole)

	if err != nil {
		return ErrBadToken
	}

	// set jwt
	cookie := &auth.AuthCookie{CookieName: "auth_token", JWT: jwt, MaxAge: 15 * 60, Secure: true, HTTPOnly: true}
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(cookie.CookieName, cookie.JWT, cookie.MaxAge, cookie.Path, cookie.Domain, cookie.Secure, cookie.HTTPOnly)
	return nil
}

// Hashes the password and stores a new user with the "user" role.
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

func GetUserService(ctx context.Context, userID string) (User, error) {
	return dbGetUser(ctx, userID)
}

// Checks the current password and hashes the new one if one was sent, then applies the changes.
func UpdateUserService(ctx context.Context, userID string, req PatchUserRequest) (User, error) {
	var hashed *string
	if req.NewPassword != nil && req.OldPassword != nil {
		current, err := dbGetPasswordHash(ctx, userID)
		if err != nil {
			return User{}, err
		}
		if current == nil {
			return User{}, ErrWrongPassword
		}

		ok, err := auth.VerifyPassword(*req.OldPassword, *current)
		if err != nil {
			return User{}, err
		}
		if !ok {
			return User{}, ErrWrongPassword
		}

		h := auth.HashPassword(*req.NewPassword)
		hashed = &h
	}

	u, err := dbUpdateUser(ctx, userID, req, hashed)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "users_email_key" {
		return User{}, ErrEmailTaken
	}

	return u, err
}
