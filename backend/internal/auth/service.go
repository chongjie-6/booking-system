package auth

import (
	"booking-system/internal/user"
	"booking-system/pkg/auth"
	"booking-system/pkg/db"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

var (
	ErrRefreshExpired     = errors.New("refresh token has expired or is revoked")
	ErrSuspiciousRefresh  = errors.New("refresh token used from a different client")
	ErrBadToken           = errors.New("jwt could not be issued")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrWrongPassword      = errors.New("current password is incorrect")
)

// ChangePasswordService checks the current password, saves the new one, revokes all the user's refresh tokens
// and issues a fresh pair for this client, in one transaction. Returns (access, refresh, err), or ErrWrongPassword.
func ChangePasswordService(ctx *gin.Context, u *auth.UserInfo, req ChangePasswordRequest) (string, string, error) {
	current, err := user.GetUserService(ctx, u.UserID)
	if err != nil {
		return "", "", err
	}
	if current.HashedPassword == nil {
		return "", "", ErrWrongPassword
	}
	err = auth.VerifyPassword(req.CurrentPassword, *current.HashedPassword)
	if errors.Is(err, auth.ErrPasswordMismatch) {
		return "", "", ErrWrongPassword
	}
	if err != nil {
		return "", "", err
	}

	var access, refresh string
	err = pgx.BeginFunc(ctx, db.Pool, func(tx pgx.Tx) error {
		if err := dbSetPasswordAndRevokeTokens(ctx, tx, u.UserID, auth.HashPassword(req.NewPassword)); err != nil {
			return err
		}
		access, refresh, err = issueUserTokens(ctx, tx, u.UserID, current.UserRole)
		return err
	})
	return access, refresh, err
}

// issueUserTokens mints an access + refresh token and saves the refresh row, tagged with the client's IP and user agent, through q (Pool or Tx).
// Takes the request context, querier, user ID and role; returns (access, refresh, err), with ErrBadToken if signing fails.
// Tokens come back only once the row is saved.
func issueUserTokens(ctx *gin.Context, q querier, userID string, userRole string) (string, string, error) {
	access, refresh, refreshExp, err := auth.MintTokens(userID, userRole)
	if err != nil {
		return "", "", ErrBadToken
	}

	userIP, userAgent := ctx.ClientIP(), ctx.Request.UserAgent()
	row := RefreshTokenRow{TokenHash: auth.HashRefreshToken(refresh), UserID: userID, UserRole: userRole, ExpiresAt: refreshExp, IsRevoked: false, UserAgent: &userAgent, IPAddress: &userIP}
	if _, err := dbInsertRefreshToken(ctx, q, row); err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

// setAuthCookies sets the access_token and refresh_token cookies. Takes the request context and both tokens; returns nothing.
// Call only after the refresh row is saved, so the client never holds a token the database doesn't know.
func setAuthCookies(ctx *gin.Context, access string, refresh string) {
	a := &auth.AuthCookie{CookieName: "access_token", JWT: access, MaxAge: int(auth.AccessTTL.Seconds()), Secure: true, HTTPOnly: true}
	r := &auth.AuthCookie{CookieName: "refresh_token", JWT: refresh, MaxAge: int(auth.RefreshTTL.Seconds()), Secure: true, HTTPOnly: true}
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie(a.CookieName, a.JWT, a.MaxAge, a.Path, a.Domain, a.Secure, a.HTTPOnly)
	ctx.SetCookie(r.CookieName, r.JWT, r.MaxAge, r.Path, r.Domain, r.Secure, r.HTTPOnly)
}

// ClearAuthCookies expires the access_token and refresh_token cookies. Takes the request context; returns nothing.
func clearAuthCookies(ctx *gin.Context) {
	ctx.SetSameSite(http.SameSiteLaxMode)
	ctx.SetCookie("access_token", "", -1, "", "", true, true)
	ctx.SetCookie("refresh_token", "", -1, "", "", true, true)
}

// RotateRefreshTokenService revokes a refresh token and issues a new access + refresh pair in one transaction, so each token works once.
// Takes the request context and raw refresh token; returns (access, refresh, err). Any error rolls back, leaving the old token as it was.
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

		access, refresh, err = issueUserTokens(ctx, tx, old.UserID, old.UserRole)
		return err
	})
	return access, refresh, err
}

// RevokeRefreshTokenService revokes a refresh token. Takes the request context and raw token;
// returns nil if it was revoked or was already invalid, so logout can be repeated safely.
func RevokeRefreshTokenService(ctx *gin.Context, refreshToken string) error {
	_, err := dbRevokeRefreshToken(ctx, db.Pool, auth.HashRefreshToken(refreshToken))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

// LoginUserService checks the email and password, then issues a saved access + refresh pair.
// Takes the request context and LoginRequest; returns (access, refresh, err), with ErrInvalidCredentials
// for an unknown email, an account with no password, or a wrong password.
func LoginUserService(ctx *gin.Context, loginRequest LoginRequest) (string, string, error) {
	u, err := dbGetUserPasswordHashFromEmail(ctx, loginRequest.Email)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", "", ErrInvalidCredentials
	}
	if err != nil {
		return "", "", err
	}
	if u.HashedPassword == nil {
		return "", "", ErrInvalidCredentials
	}

	err = auth.VerifyPassword(loginRequest.Password, *u.HashedPassword)
	if errors.Is(err, auth.ErrPasswordMismatch) {
		return "", "", ErrInvalidCredentials
	}
	if err != nil {
		return "", "", err
	}
	return issueUserTokens(ctx, db.Pool, u.UserID, u.UserRole)
}
