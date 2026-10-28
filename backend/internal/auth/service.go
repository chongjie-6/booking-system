package auth

import (
	"booking-system/internal/notification"
	"booking-system/internal/user"
	"booking-system/pkg/auth"
	"booking-system/pkg/db"
	"booking-system/pkg/geoip"
	"context"
	"crypto/rand"
	"errors"
	"net/netip"
	"sync"

	"github.com/jackc/pgx/v5"
)

var (
	ErrRefreshExpired     = errors.New("refresh token has expired or is revoked")
	ErrBadToken           = errors.New("jwt could not be issued")
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrWrongPassword      = errors.New("current password is incorrect")
)

// dummyHash is what login checks the password against when the email has no stored hash, so it does the same argon2
// work as a wrong password. Built from HashPassword on first use, so it always matches the real cost settings.
var dummyHash = sync.OnceValue(func() string { return auth.HashPassword(rand.Text()) })

// ChangePasswordService checks the current password, saves the new one, revokes all the user's refresh tokens
// and issues fresh tokens for this client, in one transaction. Takes the request context, signed-in user, request and client;
// returns (tokens, nil) for setAuthCookies, ErrWrongPassword, or the error that rolled the transaction back.
func ChangePasswordService(ctx context.Context, u *auth.UserInfo, req ChangePasswordRequest, client clientInfo) (authTokens, error) {
	current, err := user.GetUserService(ctx, u.UserID)
	if err != nil {
		return authTokens{}, err
	}
	if current.HashedPassword == nil {
		return authTokens{}, ErrWrongPassword
	}
	err = auth.VerifyPassword(req.CurrentPassword, *current.HashedPassword)
	if errors.Is(err, auth.ErrPasswordMismatch) {
		return authTokens{}, ErrWrongPassword
	}
	if err != nil {
		return authTokens{}, err
	}

	var tokens authTokens
	err = pgx.BeginFunc(ctx, db.Pool, func(tx pgx.Tx) error {
		if err := dbSetPasswordAndRevokeTokens(ctx, tx, u.UserID, auth.HashPassword(req.NewPassword)); err != nil {
			return err
		}
		tokens, err = issueUserTokens(ctx, tx, u.UserID, current.UserRole, client)
		return err
	})
	return tokens, err
}

// issueUserTokens mints an access + refresh token and saves the refresh row through q (Pool or Tx) tagged with the client's
// IP, user agent and device. The device is client.deviceID, or a new random ID if that is empty.
// Takes the request context, querier, user ID, role and client; returns (tokens, err), with ErrBadToken if signing fails.
// It doesn't set cookies: the handler passes the tokens to setAuthCookies once the row is committed.
func issueUserTokens(ctx context.Context, q querier, userID string, userRole string, client clientInfo) (authTokens, error) {
	deviceID := client.deviceID
	if deviceID == "" {
		deviceID = rand.Text()
	}

	access, refresh, refreshExp, err := auth.MintTokens(userID, userRole)
	if err != nil {
		return authTokens{}, ErrBadToken
	}

	deviceHash, refreshHash := auth.HashRefreshToken(deviceID), auth.HashRefreshToken(refresh)
	row := RefreshTokenRow{TokenHash: refreshHash, UserID: userID, UserRole: userRole, ExpiresAt: refreshExp, IsRevoked: false, UserAgent: &client.userAgent, IPAddress: &client.ip, DeviceHash: &deviceHash}
	if _, err := dbInsertRefreshToken(ctx, q, row); err != nil {
		return authTokens{}, err
	}
	return authTokens{access: access, refresh: refresh, deviceID: deviceID}, nil
}

// RotateRefreshTokenService revokes a refresh token and issues a new access + refresh pair in one transaction, so each token works once.
// Takes the request context, raw refresh token and client; returns (tokens, nil) for setAuthCookies, ErrRefreshExpired if the token is
// unknown, expired or already used, or the database error. Any error rolls back, leaving the old token as it was.
func RotateRefreshTokenService(ctx context.Context, refreshToken string, client clientInfo) (authTokens, error) {
	var tokens authTokens
	err := pgx.BeginFunc(ctx, db.Pool, func(tx pgx.Tx) error {
		old, err := dbRevokeRefreshToken(ctx, tx, auth.HashRefreshToken(refreshToken))
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrRefreshExpired
		}
		if err != nil {
			return err
		}

		tokens, err = issueUserTokens(ctx, tx, old.UserID, old.UserRole, client)
		return err
	})
	return tokens, err
}

// RevokeRefreshTokenService revokes a refresh token. Takes the request context and raw token;
// returns nil if it was revoked or was already invalid, so logout can be repeated safely.
func RevokeRefreshTokenService(ctx context.Context, refreshToken string) error {
	_, err := dbRevokeRefreshToken(ctx, db.Pool, auth.HashRefreshToken(refreshToken))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	return err
}

// LoginUserService checks the email and password, then issues tokens. If the device is new to the user, or the IP is
// on a network and in a country they haven't used (see isKnownIP), it also sends a suspicious-activity notification in the background.
// Takes the request context, LoginRequest and client; returns (tokens, nil) for setAuthCookies, ErrInvalidCredentials for an unknown
// email, an account with no password, or a wrong password, or the database error.
func LoginUserService(ctx context.Context, loginRequest LoginRequest, client clientInfo) (authTokens, error) {
	u, err := dbGetUserPasswordHashFromEmail(ctx, loginRequest.Email)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return authTokens{}, err
	}

	// Unknown emails and accounts with no password still run argon2 (against dummyHash),
	// so response time doesn't reveal which emails are registered.
	hash := dummyHash()
	if u.HashedPassword != nil {
		hash = *u.HashedPassword
	}
	err = auth.VerifyPassword(loginRequest.Password, hash)
	if u.HashedPassword == nil || errors.Is(err, auth.ErrPasswordMismatch) {
		return authTokens{}, ErrInvalidCredentials
	}
	if err != nil {
		return authTokens{}, err
	}

	// Alert the user when they log in from a device or network they haven't used before.
	// A new device alerts anyway, so the IP history is only checked for a known one.
	known := false
	if client.deviceID != "" {
		if known, err = dbHasDevice(ctx, u.UserID, auth.HashRefreshToken(client.deviceID)); err != nil {
			return authTokens{}, err
		}
	}
	if known {
		if known, err = isKnownIP(ctx, u.UserID, client.ip); err != nil {
			return authTokens{}, err
		}
	}

	tokens, err := issueUserTokens(ctx, db.Pool, u.UserID, u.UserRole, client)
	if err != nil {
		return authTokens{}, err
	}
	if !known {
		notification.SendSuspiciousActivityNotification(u.UserID, client.ip, client.userAgent)
	}
	return tokens, nil
}

// isKnownIP reports whether the user has been issued a token before on a nearby network or from the same country. Takes a context, user ID and IP; returns (known, err).
func isKnownIP(ctx context.Context, userID string, ip string) (bool, error) {
	seen, err := dbGetSeenIPs(ctx, userID)
	if err != nil {
		return false, err
	}
	return onSeenNetwork(ip, seen) || inSeenCountry(ip, seen, geoip.Country), nil
}

// inSeenCountry reports whether ip is in the same country as any seen IP. Takes the IP, the seen IPs, and a lookup
// that returns an ISO country code; an unknown country ("") never matches, so private IPs fall back to onSeenNetwork.
func inSeenCountry(ip string, seen []string, country func(netip.Addr) string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	want := country(addr.Unmap())
	if want == "" {
		return false
	}
	for _, s := range seen {
		if a, err := netip.ParseAddr(s); err == nil && country(a.Unmap()) == want {
			return true
		}
	}
	return false
}

// onSeenNetwork reports whether ip shares a /24 (IPv4) or /48 (IPv6) with any seen IP,
// so a new address on the same home or office network still counts as known. An unparseable ip matches nothing.
func onSeenNetwork(ip string, seen []string) bool {
	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}
	addr = addr.Unmap()
	bits := 48
	if addr.Is4() {
		bits = 24
	}
	network, _ := addr.Prefix(bits)
	for _, s := range seen {
		if a, err := netip.ParseAddr(s); err == nil && network.Contains(a.Unmap()) {
			return true
		}
	}
	return false
}
