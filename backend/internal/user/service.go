package user

import (
	"booking-system/pkg/auth"
	"booking-system/pkg/db"
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

var (
	ErrEmailTaken    = errors.New("email already registered")
	ErrWrongPassword = errors.New("current password is incorrect")
	ErrBadToken      = errors.New("jwt could not be issued")

	ErrRefreshExpired    = errors.New("refresh token has expired or is revoked")
	ErrSuspiciousRefresh = errors.New("refresh token used from a different client")
)

// Mints access + refresh tokens and builds the refresh row to store. Doesn't touch the response.
func MintUserTokens(ctx *gin.Context, userID string, userRole string) (string, string, RefreshTokenRow, error) {
	access, refresh, refreshExp, err := auth.MintTokens(userID, userRole)
	if err != nil {
		return "", "", RefreshTokenRow{}, ErrBadToken
	}

	userIP, userAgent := ctx.ClientIP(), ctx.Request.UserAgent()
	row := RefreshTokenRow{TokenHash: auth.HashRefreshToken(refresh), UserID: userID, UserRole: userRole, ExpiresAt: refreshExp, IsRevoked: false, UserAgent: &userAgent, IPAddress: &userIP}
	return access, refresh, row, nil
}

// Call only after the refresh row is saved, so the client never holds a token the database doesn't know.
func SetAuthCookies(ctx *gin.Context, access string, refresh string) {
	a := &auth.AuthCookie{CookieName: "access_token", JWT: access, MaxAge: int(auth.AccessTTL.Seconds()), Secure: true, HTTPOnly: true}
	r := &auth.AuthCookie{CookieName: "refresh_token", JWT: refresh, MaxAge: int(auth.RefreshTTL.Seconds()), Secure: true, HTTPOnly: true}
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(a.CookieName, a.JWT, a.MaxAge, a.Path, a.Domain, a.Secure, a.HTTPOnly)
	ctx.SetCookie(r.CookieName, r.JWT, r.MaxAge, r.Path, r.Domain, r.Secure, r.HTTPOnly)
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

// Swaps a refresh token for a new access + refresh pair in one transaction, so each refresh token works once.
// Any error rolls back, leaving the old token as it was.
func RotateRefreshTokenService(ctx *gin.Context, refreshToken string) (string, string, error) {
	var access, refresh string
	err := pgx.BeginFunc(ctx, db.Pool, func(tx pgx.Tx) error {
		old, err := dbRevokeRefreshToken(ctx, tx, auth.HashRefreshToken(refreshToken))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRefreshExpired
		}
		if err != nil {
			return err
		}

		//TODO implement notifications to user email about suspicious login
		if old.IPAddress == nil || *old.IPAddress != ctx.ClientIP() ||
			old.UserAgent == nil || *old.UserAgent != ctx.Request.UserAgent() {
			return ErrSuspiciousRefresh
		}

		a, r, row, err := MintUserTokens(ctx, old.UserID, old.UserRole)
		if err != nil {
			return err
		}
		if _, err := dbInsertRefreshToken(ctx, tx, row); err != nil {
			return err
		}
		access, refresh = a, r
		return nil
	})
	return access, refresh, err
}
