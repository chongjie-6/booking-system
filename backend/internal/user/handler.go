package user

import (
	"booking-system/pkg/auth"
	"booking-system/pkg/response"
	"errors"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
)

// Ping returns pong
func Ping(c *gin.Context) {
	response.Ok(c, gin.H{"message": "pong"}, nil)
}

// Creates a user; a duplicate email returns 409
func CreateUser(c *gin.Context) {
	var req CreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(response.BadRequest(err))
		return
	}

	// create user in database
	newUser, err := CreateUserService(c, req)

	if errors.Is(err, ErrEmailTaken) {
		c.Error(response.NewError(http.StatusConflict, "user already exists"))
		return
	}
	if err != nil {
		c.Error(response.ErrInternalServerError)
		return
	}

	//attach cookies to response (must be in headers)
	err = AttachUserCookies(c, newUser.UserID, newUser.UserRole)
	if errors.Is(err, ErrBadToken) {
		log.Printf("create user: %v", err)
		c.Error(response.NewError(http.StatusInternalServerError, "account created, but we couldn't sign you in. Please log in"))
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

// Updates the signed-in user's details; a taken email returns 409
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
	if errors.Is(err, ErrWrongPassword) {
		c.Error(response.NewError(http.StatusForbidden, "password is incorrect"))
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
