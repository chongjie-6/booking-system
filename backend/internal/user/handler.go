package user

import (
	"booking-system/pkg/middleware"
	"booking-system/pkg/response"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5"
)

// Ping is a health check for the user routes. Takes the request; responds 200 with {"message": "pong"}.
func Ping(w http.ResponseWriter, r *http.Request) {
	response.Ok(w, map[string]string{"message": "pong"}, nil)
}

// GetUser returns the signed-in user's details. Takes the request (with "user" set by middleware.Auth);
// responds 200 with the user, 404 if they no longer exist, or 500.
func GetUser(w http.ResponseWriter, r *http.Request) {
	user := middleware.User(r)

	u, err := GetUserService(r.Context(), user.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		response.Error(w, response.ErrNotFound)
		return
	}

	if err != nil {
		response.Error(w, response.ErrInternalServerError)
		return
	}

	response.Ok(w, u, nil)
}

// PatchUser updates the signed-in user from a PatchUserRequest JSON body. Takes the request (with "user" set by middleware.Auth);
// responds 200 with the updated user, or 400 bad body, 404 no user, 409 email taken, 500.
func PatchUser(w http.ResponseWriter, r *http.Request) {
	user := middleware.User(r)

	var req PatchUserRequest
	if err := response.Bind(r, &req); err != nil {
		response.Error(w, err)
		return
	}

	u, err := UpdateUserService(r.Context(), user.UserID, req)
	if errors.Is(err, ErrEmailTaken) {
		response.Error(w, response.NewError(http.StatusConflict, "email already in use"))
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		response.Error(w, response.ErrNotFound)
		return
	}
	if err != nil {
		response.Error(w, response.ErrInternalServerError)
		return
	}

	response.Ok(w, u, nil)
}
