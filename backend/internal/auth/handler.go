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
	newUser, err := user.CreateUserService(c, req)
	if errors.Is(err, user.ErrEmailTaken) {
		c.Error(response.NewError(http.StatusConflict, "user already exists"))
		return
	}
	if err != nil {
		c.Error(response.ErrInternalServerError)
		return
	}

	access, refresh, err := issueUserTokens(c, db.Pool, newUser.UserID, newUser.UserRole)
	if err != nil {
		log.Printf("create user: %v", err)
		c.Error(response.NewError(http.StatusInternalServerError, "account created, but we couldn't sign you in. Please log in"))
		return
	}

	setAuthCookies(c, access, refresh)
	response.Created(c, newUser)
}

// RefreshToken swaps the refresh_token cookie for a new access + refresh pair.
// Takes the request context; responds 200 with fresh auth cookies, or 401 if the token is invalid or used from another client.
func RefreshToken(c *gin.Context) {
	refreshToken, err := c.Cookie("refresh_token")
	if err != nil {
		c.Error(response.ErrUnauthorized)
		return
	}

	access, refresh, err := RotateRefreshTokenService(c, refreshToken)
	if errors.Is(err, ErrRefreshExpired) || errors.Is(err, ErrSuspiciousRefresh) {
		c.Error(response.ErrUnauthorized)
		return
	}
	if err != nil {
		log.Printf("refresh token: %v", err)
		c.Error(response.ErrInternalServerError)
		return
	}

	setAuthCookies(c, access, refresh)
	response.Ok(c, nil, nil)
}

// Logout revokes the refresh_token cookie and clears both auth cookies.
// Takes the request context; responds 200 even with no cookie or an already-revoked token, or 500 if the revoke fails.
func Logout(c *gin.Context) {
	if refreshToken, err := c.Cookie("refresh_token"); err == nil {
		if err := RevokeRefreshTokenService(c, refreshToken); err != nil {
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

	access, refresh, err := ChangePasswordService(c, u, req)
	if errors.Is(err, ErrWrongPassword) {
		c.Error(response.NewError(http.StatusForbidden, "password is incorrect"))
		return
	}
	if err != nil {
		log.Printf("change password: %v", err)
		c.Error(response.ErrInternalServerError)
		return
	}

	setAuthCookies(c, access, refresh)
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

	access, refresh, err := LoginUserService(c, req)
	if errors.Is(err, ErrInvalidCredentials) {
		c.Error(response.ErrUnauthorized)
		return
	}
	if err != nil {
		log.Printf("login: %v", err)
		c.Error(response.ErrInternalServerError)
		return
	}

	setAuthCookies(c, access, refresh)
	response.Ok(c, nil, nil)
}
