package user

import (
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
		c.Abort()
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	// max=72 counts characters, bcrypt counts bytes, so emoji/accents can still overflow
	if errors.Is(err, bcrypt.ErrPasswordTooLong) {
		c.Error(response.NewError(http.StatusBadRequest, "password is too long"))
		c.Abort()
		return
	}
	if err != nil {
		c.Error(response.ErrInternalServerError)
		c.Abort()
		return
	}
	hashed := string(hash)

	newUser, err := dbCreateUser(c, User{
		UserRole:       "user",
		HashedPassword: &hashed,
		FirstName:      req.FirstName,
		LastName:       req.LastName,
		Email:          req.Email,
	})
	if errors.Is(err, ErrEmailTaken) {
		c.Error(response.NewError(http.StatusConflict, "user already exists"))
		c.Abort()
		return
	}
	if err != nil {
		c.Error(response.ErrInternalServerError)
		c.Abort()
		return
	}

	response.Created(c, newUser)
}

type GetUserRequest struct {
	UserID string `json:"user_id" binding:"required,uuid"`
}

// Get a users' details
func GetUser(c *gin.Context) {
	var req GetUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.Error(response.BadRequest(err))
		c.Abort()
		return
	}

	u, err := dbGetUser(c, req.UserID)
	if errors.Is(err, pgx.ErrNoRows) {
		c.Error(response.ErrNotFound)
		c.Abort()
		return
	}
	if err != nil {
		c.Error(response.ErrInternalServerError)
		c.Abort()
		return
	}

	response.Ok(c, u, nil)
}
