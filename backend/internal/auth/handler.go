package auth

import (
	"booking-system/internal/user"
	"booking-system/pkg/auth"
	"booking-system/pkg/db"
	"booking-system/pkg/middleware"
	"booking-system/pkg/response"
	"errors"
	"log"
	"net/http"
	"time"
)

// cookie returns the named cookie's value, or "" if the request doesn't have it.
func cookie(r *http.Request, name string) string {
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

// setCookie sets a secure, HTTP-only, SameSite=Lax cookie. A negative maxAge deletes it. Takes the writer and cookie fields; returns nothing.
func setCookie(w http.ResponseWriter, name, value, path string, maxAge time.Duration) {
	http.SetCookie(w, &http.Cookie{Name: name, Value: value, Path: path, MaxAge: int(maxAge.Seconds()),
		Secure: true, HttpOnly: true, SameSite: http.SameSiteLaxMode})
}

// clientFrom reads the client's IP, user agent and device_id cookie ("" if missing) for the service layer.
func clientFrom(r *http.Request) clientInfo {
	return clientInfo{ip: middleware.ClientIP(r), userAgent: r.UserAgent(), deviceID: cookie(r, "device_id")}
}

// setAuthCookies sets the access_token, refresh_token and device_id cookies. Takes the writer and tokens; returns nothing.
// Call only after the refresh row is committed, so the client never holds a token the database doesn't know.
func setAuthCookies(w http.ResponseWriter, t authTokens) {
	setCookie(w, "access_token", t.access, "/", auth.AccessTTL)
	setCookie(w, "refresh_token", t.refresh, "/auth", auth.RefreshTTL)
	setCookie(w, "device_id", t.deviceID, "/auth", auth.DeviceTTL)
}

// clearAuthCookies expires the access_token and refresh_token cookies. Takes the writer; returns nothing.
func clearAuthCookies(w http.ResponseWriter) {
	setCookie(w, "access_token", "", "/", -1)
	setCookie(w, "refresh_token", "", "/auth", -1)
}

// Ping is a health check for the auth routes. Takes the request; responds 200 with {"message": "pong"}.
func Ping(w http.ResponseWriter, r *http.Request) {
	response.Ok(w, map[string]string{"message": "pong"}, nil)
}

// Register creates an account from a CreateUserRequest JSON body and signs the user in.
// Takes the request; responds 201 with the new user and sets auth cookies, or 400/409/500 on error.
func Register(w http.ResponseWriter, r *http.Request) {
	var req user.CreateUserRequest
	if err := response.Bind(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	newUser, err := user.CreateUserService(r.Context(), req)
	if errors.Is(err, user.ErrEmailTaken) {
		response.Error(w, response.NewError(http.StatusConflict, "user already exists"))
		return
	}
	if err != nil {
		response.Error(w, response.ErrInternalServerError)
		return
	}

	tokens, err := issueUserTokens(r.Context(), db.Pool, newUser.UserID, newUser.UserRole, clientFrom(r))
	if err != nil {
		log.Printf("create user: %v", err)
		response.Error(w, response.NewError(http.StatusInternalServerError, "account created, but we couldn't sign you in. Please log in"))
		return
	}
	setAuthCookies(w, tokens)
	response.Created(w, newUser)
}

// RefreshToken swaps the refresh_token cookie for a new access + refresh pair.
// Takes the request; responds 200 with fresh auth cookies, or 401 if the token is unknown, expired or already used.
func RefreshToken(w http.ResponseWriter, r *http.Request) {
	refreshToken := cookie(r, "refresh_token")
	if refreshToken == "" {
		response.Error(w, response.ErrUnauthorized)
		return
	}

	tokens, err := RotateRefreshTokenService(r.Context(), refreshToken, clientFrom(r))
	if errors.Is(err, ErrRefreshExpired) {
		response.Error(w, response.ErrUnauthorized)
		return
	}
	if err != nil {
		log.Printf("refresh token: %v", err)
		response.Error(w, response.ErrInternalServerError)
		return
	}

	setAuthCookies(w, tokens)
	response.Ok(w, nil, nil)
}

// Logout revokes the refresh_token cookie and clears both auth cookies.
// Takes the request; responds 200 even with no cookie or an already-revoked token, or 500 if the revoke fails.
func Logout(w http.ResponseWriter, r *http.Request) {
	if refreshToken := cookie(r, "refresh_token"); refreshToken != "" {
		if err := RevokeRefreshTokenService(r.Context(), refreshToken); err != nil {
			log.Printf("logout: %v", err)
			response.Error(w, response.ErrInternalServerError)
			return
		}
	}
	clearAuthCookies(w)
	response.Ok(w, nil, nil)

}

// ChangePassword sets a new password, signs out every other session and re-signs this one.
// Takes the request (with "user" set by middleware.Auth); responds 200 with fresh cookies, or 400/403/500.
func ChangePassword(w http.ResponseWriter, r *http.Request) {
	u := middleware.User(r)

	var req ChangePasswordRequest
	if err := response.Bind(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	tokens, err := ChangePasswordService(r.Context(), u, req, clientFrom(r))
	if errors.Is(err, ErrWrongPassword) {
		response.Error(w, response.NewError(http.StatusForbidden, "password is incorrect"))
		return
	}
	if err != nil {
		log.Printf("change password: %v", err)
		response.Error(w, response.ErrInternalServerError)
		return
	}

	setAuthCookies(w, tokens)
	response.Ok(w, nil, nil)
}

// ForgotPassword emails a reset link if the ForgotPasswordRequest email has an account.
// Takes the request; responds 200 whether it does or not, so the response doesn't reveal which emails are registered, or 400/500.
func ForgotPassword(w http.ResponseWriter, r *http.Request) {
	var req ForgotPasswordRequest
	if err := response.Bind(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	if err := ForgotPasswordService(r.Context(), req.Email); err != nil {
		log.Printf("forgot password: %v", err)
		response.Error(w, response.ErrInternalServerError)
		return
	}
	response.Ok(w, nil, nil)
}

// ResetPassword sets a new password from a ResetPasswordRequest JSON body and signs out every session.
// Takes the request; responds 200 and clears this browser's auth cookies, 400 if the link is invalid, expired or used, or 500.
func ResetPassword(w http.ResponseWriter, r *http.Request) {
	var req ResetPasswordRequest
	if err := response.Bind(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	err := ResetPasswordService(r.Context(), req)
	if errors.Is(err, ErrResetTokenInvalid) {
		response.Error(w, response.NewError(http.StatusBadRequest, "this reset link is invalid or has expired. Request a new one"))
		return
	}
	if err != nil {
		log.Printf("reset password: %v", err)
		response.Error(w, response.ErrInternalServerError)
		return
	}
	clearAuthCookies(w)
	response.Ok(w, nil, nil)
}

// Login signs the user in from a LoginRequest JSON body.
// Takes the request; responds 200 with auth cookies, 400 if the body is invalid, 401 for a wrong email or password, or 500 on error.
func Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := response.Bind(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	tokens, err := LoginUserService(r.Context(), req, clientFrom(r))
	if errors.Is(err, ErrInvalidCredentials) {
		response.Error(w, response.ErrUnauthorized)
		return
	}
	if err != nil {
		log.Printf("login: %v", err)
		response.Error(w, response.ErrInternalServerError)
		return
	}

	setAuthCookies(w, tokens)
	response.Ok(w, nil, nil)
}
