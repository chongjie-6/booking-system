package middleware

import (
	"booking-system/pkg/auth"
	"booking-system/pkg/response"
	"context"
	"net/http"
)

type userKey struct{}

// Auth wraps next so it only runs when the access_token cookie is valid; otherwise it responds 401.
// Takes the handler; returns it wrapped, with the token's *auth.UserInfo readable through User.
func Auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("access_token")
		if err != nil {
			response.Error(w, response.ErrUnauthorized)
			return
		}

		claims, err := auth.ParseJWT(cookie.Value)
		if err != nil || claims.User == nil {
			response.Error(w, response.ErrUnauthorized)
			return
		}

		next(w, r.WithContext(context.WithValue(r.Context(), userKey{}, claims.User)))
	}
}

// User returns the signed-in user that Auth stored on the request. Takes the request; returns nil outside an Auth-wrapped handler.
func User(r *http.Request) *auth.UserInfo {
	u, _ := r.Context().Value(userKey{}).(*auth.UserInfo)
	return u
}
