package user

import (
	"booking-system/pkg/auth"
	"booking-system/pkg/response"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// Ping is a health check for the user routes. Takes the request context; responds 200 with {"message": "pong"}.
func Ping(c *gin.Context) {
	response.Ok(c, gin.H{"message": "pong"}, nil)
}

// GetUser returns the signed-in user's details. Takes the request context (with "user" set by middleware.Auth);
// responds 200 with the user, 404 if they no longer exist, or 500.
func GetUser(c *gin.Context) {
	user, _ := c.MustGet("user").(*auth.UserInfo)

	u, err := GetUserService(c, user.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.Error(response.ErrNotFound)
		return
	}

	if err != nil {
		c.Error(response.ErrInternalServerError)
		return
	}

	response.Ok(c, u, nil)
}

// PatchUser updates the signed-in user from a PatchUserRequest JSON body. Takes the request context (with "user" set by middleware.Auth);
// responds 200 with the updated user, or 400 bad body, 404 no user, 409 email taken, 500.
func PatchUser(c *gin.Context) {
	user, _ := c.MustGet("user").(*auth.UserInfo)

	var req PatchUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(response.BadRequest(err))
		return
	}

	u, err := UpdateUserService(c, user.UserID, req)
	if errors.Is(err, ErrEmailTaken) {
		c.Error(response.NewError(http.StatusConflict, "email already in use"))
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		c.Error(response.ErrNotFound)
		return
	}
	if err != nil {
		c.Error(response.ErrInternalServerError)
		return
	}

	response.Ok(c, u, nil)
}
