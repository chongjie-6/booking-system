package response

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Success bool      `json:"success"`
	Data    any       `json:"data,omitempty"`
	Error   *AppError `json:"error,omitempty"`
	Meta    *Meta     `json:"meta,omitempty"`
}

type AppError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Status  int    `json:"-"`
}

type Meta struct {
	Page       int `json:"page"`
	PerPage    int `json:"perPage"`
	Total      int `json:"total"`
	TotalPages int `json:"totalPages"`
}

// 400 Error, user sent invalid request
var ErrBadRequest = &AppError{Status: http.StatusBadRequest, Code: "BAD_REQUEST", Message: "invalid request"}

// 401 Error, user lacks valid credentials
var ErrUnauthorized = &AppError{Status: http.StatusUnauthorized, Code: "UNAUTHORIZED", Message: "missing or invalid credentials"}

// 403 Error, user not allowed to see this resource
var ErrForbidden = &AppError{Status: http.StatusForbidden, Code: "FORBIDDEN", Message: "you are not allowed to access this resource"}

// 404 Error, resource not found
var ErrNotFound = &AppError{Status: http.StatusNotFound, Code: "NOT_FOUND", Message: "resource not found"}

// 429, too many requests
var ErrTooManyRequests = &AppError{Status: http.StatusTooManyRequests, Code: "TOO_MANY_REQUESTS", Message: "rate limit exceeded"}

// 500 Error, something went wrong with servers
var ErrInternalServerError = &AppError{Status: http.StatusInternalServerError, Code: "INTERNAL_SERVER_ERROR", Message: "internal server error"}

func (e *AppError) Error() string {
	return e.Message
}

// 200, successfull and returned data
func Ok(c *gin.Context, data any, meta *Meta) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// 201, successfully created a resource
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

func ErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// If no error, continue with main func
		if len(c.Errors) == 0 {
			return
		}

		appError, ok := errors.AsType[*AppError](c.Errors.Last().Err)
		if !ok {
			appError = ErrInternalServerError
		}

		c.AbortWithStatusJSON(appError.Status, &Response{Success: false, Error: appError})
	}
}
