package user

import (
	"booking-system/pkg/auth"
	"context"
	"crypto/sha256"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrEmailTaken    = errors.New("email already registered")
	ErrWrongPassword = errors.New("current password is incorrect")
	ErrBadToken      = errors.New("jwt could not be issued")

	ErrRefreshExpired    = errors.New("refresh token has expired or is revoked")
	ErrSuspiciousRefresh = errors.New("refresh token used from a different client")
)

func AttachUserCookies(ctx *gin.Context, userID string, userRole string) error {
	// issue new access + refresh
	access, refresh, err := auth.MintTokens(userID, userRole)

	if err != nil {
		return ErrBadToken
	}

	// set access and refresh cookies
	a := &auth.AuthCookie{CookieName: "access_token", JWT: access, MaxAge: 15 * 60, Secure: true, HTTPOnly: true}
	r := &auth.AuthCookie{CookieName: "refresh_token", JWT: refresh, MaxAge: 60 * 60 * 24 * 7, Secure: true, HTTPOnly: true}
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(a.CookieName, a.JWT, a.MaxAge, a.Path, a.Domain, a.Secure, a.HTTPOnly)
	ctx.SetCookie(r.CookieName, r.JWT, r.MaxAge, r.Path, r.Domain, r.Secure, r.HTTPOnly)
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

// fetches the database and validates against token stored in the database
func ValidateRefreshTokenService(ctx *gin.Context, refreshToken string) (RefreshTokenRow, error) {
	// hash the refreshToken and then find corresponding row in the database
	hashedRefresh := auth.HashRefreshToken(refreshToken)
	refresh, err := dbGetRefreshToken(ctx, hashedRefresh)
	if err != nil {
		return RefreshTokenRow{}, err
	}

	if refresh.IsRevoked || time.Now().After(refresh.ExpiresAt) {
		return RefreshTokenRow{}, ErrRefreshExpired
	}

	//TODO implement notifications to user email about suspicious login
	if refresh.IPAddress == nil || *refresh.IPAddress != ctx.ClientIP() ||
		refresh.UserAgent == nil || *refresh.UserAgent != ctx.Request.UserAgent() {
		return RefreshTokenRow{}, ErrSuspiciousRefresh
	}

	return refresh, nil
}

// rotates refresh token hash, delete + insert
func RotateRefreshTokenService(ctx context.Context, refreshToken RefreshTokenRow, oldRefreshToken string) (bool, error) {
	// insert new refresh token
	_, createErr := dbCreateRefreshToken(ctx, refreshToken)

	if createErr != nil {
		return false, createErr
	}

	oldRefreshHash := auth.HashRefreshToken(oldRefreshToken)
	_, delErr := dbSoftDeleteRefreshToken(ctx, oldRefreshHash)
	if delErr != nil {
		return false, delErr
	}
	return true, nil
}
