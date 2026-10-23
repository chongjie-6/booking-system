package user

import (
	"booking-system/pkg/auth"
	"booking-system/pkg/response"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

// Ping returns pong
func Ping(c *gin.Context) {
	response.Ok(c, gin.H{"message": "pong"}, nil)
}

// CreateUserRequest is what a client may send to sign up. bcrypt rejects passwords over 72 bytes.
type CreateUserRequest struct {
	FirstName string `json:"firstName" binding:"required"`
	LastName  string `json:"lastName" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8,max=72"`
}

// Creates a user; a duplicate email returns 409
func CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(response.BadRequest(err))
		return
	}

	newUser, err := CreateUserService(c, req)
	if errors.Is(err, bcrypt.ErrPasswordTooLong) {
		c.Error(response.NewError(http.StatusBadRequest, "password is too long"))
		return
	}
	if errors.Is(err, ErrEmailTaken) {
		c.Error(response.NewError(http.StatusConflict, "user already exists"))
		return
	}
	if err != nil {
		c.Error(response.ErrInternalServerError)
		return
	}

	response.Created(c, newUser)
}

// Get a users' details
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

// PatchUserRequest holds the fields a user may change; omitted fields stay as they are.
type PatchUserRequest struct {
	FirstName *string `json:"firstName" binding:"omitempty,min=1"`
	LastName  *string `json:"lastName" binding:"omitempty,min=1"`
	Email     *string `json:"email" binding:"omitempty,email"`
	Password  *string `json:"password" binding:"omitempty,min=8,max=72"`
}

// Updates the signed-in user's details; a taken email returns 409
func PatchUser(c *gin.Context) {
	user, _ := c.MustGet("user").(*auth.UserInfo)

	var req PatchUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(response.BadRequest(err))
		return
	}

	u, err := UpdateUserService(c, user.UserID, req)
	if errors.Is(err, bcrypt.ErrPasswordTooLong) {
		c.Error(response.NewError(http.StatusBadRequest, "password is too long"))
		return
	}
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
