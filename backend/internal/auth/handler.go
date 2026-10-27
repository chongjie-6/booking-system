package auth

import (
	"booking-system/internal/user"
	"booking-system/pkg/auth"
	"booking-system/pkg/db"
	"booking-system/pkg/response"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

// clientFrom reads the client's IP, user agent and device_id cookie ("" if missing) for the service layer.
func clientFrom(c *gin.Context) clientInfo {
	deviceID, _ := c.Cookie("device_id")
	return clientInfo{ip: c.ClientIP(), userAgent: c.Request.UserAgent(), deviceID: deviceID}
}

// setAuthCookies sets the access_token, refresh_token and device_id cookies. Takes the request context and tokens; returns nothing.
// Call only after the refresh row is committed, so the client never holds a token the database doesn't know.
func setAuthCookies(c *gin.Context, t authTokens) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", t.access, int(auth.AccessTTL.Seconds()), "", "", true, true)
	c.SetCookie("refresh_token", t.refresh, int(auth.RefreshTTL.Seconds()), "/auth", "", true, true)
	c.SetCookie("device_id", t.deviceID, int(auth.DeviceTTL.Seconds()), "/auth", "", true, true)
}

// clearAuthCookies expires the access_token and refresh_token cookies. Takes the request context; returns nothing.
func clearAuthCookies(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", "", -1, "", "", true, true)
	c.SetCookie("refresh_token", "", -1, "/auth", "", true, true)
}

// Ping is a health check for the auth routes. Takes the request context; responds 200 with {"message": "pong"}.
func Ping(c *gin.Context) {
	response.Ok(c, gin.H{"message": "pong"}, nil)
}

// Register creates an account from a CreateUserRequest JSON body and signs the user in.
// Takes the request context; responds 201 with the new user and sets auth cookies, or 400/409/500 on error.
func Register(c *gin.Context) {
	var req user.CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(response.BadRequest(err))
		return
	}
	newUser, err := user.CreateUserService(c.Request.Context(), req)
	if errors.Is(err, user.ErrEmailTaken) {
		c.Error(response.NewError(http.StatusConflict, "user already exists"))
		return
	}
	if err != nil {
		c.Error(response.ErrInternalServerError)
		return
	}

	tokens, err := issueUserTokens(c.Request.Context(), db.Pool, newUser.UserID, newUser.UserRole, clientFrom(c))
	if err != nil {
		log.Printf("create user: %v", err)
		c.Error(response.NewError(http.StatusInternalServerError, "account created, but we couldn't sign you in. Please log in"))
		return
	}
	setAuthCookies(c, tokens)
	response.Created(c, newUser)
}

// RefreshToken swaps the refresh_token cookie for a new access + refresh pair.
// Takes the request context; responds 200 with fresh auth cookies, or 401 if the token is unknown, expired or already used.
func RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.Error(response.ErrUnauthorized)
		return
	}

	tokens, err := RotateRefreshTokenService(c.Request.Context(), refreshToken, clientFrom(c))
	if errors.Is(err, ErrRefreshExpired) {
		c.Error(response.ErrUnauthorized)
		return
	}
	if err != nil {
		log.Printf("refresh token: %v", err)
		c.Error(response.ErrInternalServerError)
		return
	}

	setAuthCookies(c, tokens)
	response.Ok(c, nil, nil)
}

// Logout revokes the refresh_token cookie and clears both auth cookies.
// Takes the request context; responds 200 even with no cookie or an already-revoked token, or 500 if the revoke fails.
func Logout(c *gin.Context) {
	if refreshToken, err := c.Cookie("refresh_token"); err == nil {
		if err := RevokeRefreshTokenService(c.Request.Context(), refreshToken); err != nil {
			log.Printf("logout: %v", err)
			c.Error(response.ErrInternalServerError)
			return
		}
	}
	clearAuthCookies(c)
	response.Ok(c, nil, nil)

}

// ChangePassword sets a new password, signs out every other session and re-signs this one.
// Takes the request context (with "user" set by middleware.Auth); responds 200 with fresh cookies, or 400/403/500.
func ChangePassword(c *gin.Context) {
	u, _ := c.MustGet("user").(*auth.UserInfo)

	var req ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(response.BadRequest(err))
		return
	}

	tokens, err := ChangePasswordService(c.Request.Context(), u, req, clientFrom(c))
	if errors.Is(err, ErrWrongPassword) {
		c.Error(response.NewError(http.StatusForbidden, "password is incorrect"))
		return
	}
	if err != nil {
		log.Printf("change password: %v", err)
		c.Error(response.ErrInternalServerError)
		return
	}

	setAuthCookies(c, tokens)
	response.Ok(c, nil, nil)
}

// Login signs the user in from a LoginRequest JSON body.
// Takes the request context; responds 200 with auth cookies, 400 if the body is invalid, 401 for a wrong email or password, or 500 on error.
func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindBodyWithJSON(&req); err != nil {
		c.Error(response.BadRequest(err))
		return
	}

	tokens, err := LoginUserService(c.Request.Context(), req, clientFrom(c))
	if errors.Is(err, ErrInvalidCredentials) {
		c.Error(response.ErrUnauthorized)
		return
	}
	if err != nil {
		log.Printf("login: %v", err)
		c.Error(response.ErrInternalServerError)
		return
	}

	setAuthCookies(c, tokens)
	response.Ok(c, nil, nil)
}
