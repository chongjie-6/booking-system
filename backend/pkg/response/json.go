package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// init makes validation errors use the JSON field name (firstName) instead of the Go one (FirstName).
// Runs once when the package loads; no input or output.
func init() {
	if v, ok := binding.Validator.Engine().(*validator.Validate); ok {
		v.RegisterTagNameFunc(func(f reflect.StructField) string {
			return strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
		})
	}
}

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
var ErrBadRequest = NewError(http.StatusBadRequest, "invalid request")

// 401 Error, user lacks valid credentials
var ErrUnauthorized = NewError(http.StatusUnauthorized, "missing or invalid credentials")

// 403 Error, user not allowed to see this resource
var ErrForbidden = NewError(http.StatusForbidden, "you are not allowed to access this resource")

// 404 Error, resource not found
var ErrNotFound = NewError(http.StatusNotFound, "resource not found")

// 409 Error, request clashes with existing data
var ErrConflict = NewError(http.StatusConflict, "resource already exists")

// 429, too many requests
var ErrTooManyRequests = NewError(http.StatusTooManyRequests, "rate limit exceeded")

// 500 Error, something went wrong with servers
var ErrInternalServerError = NewError(http.StatusInternalServerError, "internal server error, please try again later")

// NewError builds an AppError for an HTTP status. Code comes from the status text, e.g. 409 -> "CONFLICT".
// Takes the status and message; returns the *AppError.
func NewError(status int, message string) *AppError {
	code := strings.ToUpper(strings.ReplaceAll(http.StatusText(status), " ", "_"))
	return &AppError{Status: status, Code: code, Message: message}
}

// Error makes AppError satisfy the error interface. Takes nothing; returns the message.
func (e *AppError) Error() string {
	return e.Message
}

// BadRequest turns a ShouldBindJSON error into a 400 that tells the user what to fix.
// Takes the bind error; returns the *AppError.
func BadRequest(err error) *AppError {
	return NewError(http.StatusBadRequest, bindMessage(err))
}

// bindMessage turns a JSON bind error (empty body, bad JSON, wrong type, failed validation) into a readable message.
// Takes the error; returns the message.
func bindMessage(err error) string {
	if errors.Is(err, io.EOF) {
		return "request body is empty"
	}
	if errors.Is(err, io.ErrUnexpectedEOF) {
		return "request body is not valid JSON"
	}
	if _, ok := errors.AsType[*json.SyntaxError](err); ok {
		return "request body is not valid JSON"
	}
	if te, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
		if te.Field == "" {
			return "request body must be a JSON object"
		}
		return fmt.Sprintf("%s must be a %s", te.Field, te.Type)
	}
	if ve, ok := errors.AsType[validator.ValidationErrors](err); ok {
		msgs := make([]string, len(ve))
		for i, fe := range ve {
			msgs[i] = fieldMessage(fe)
		}
		return strings.Join(msgs, "; ")
	}
	return ErrBadRequest.Message
}

// fieldMessage describes one failed validation rule, e.g. "password must be at least 8 characters".
// Takes the field error; returns the message.
func fieldMessage(fe validator.FieldError) string {
	f := fe.Field()
	unit := ""
	if fe.Kind() == reflect.String {
		unit = " characters"
	}
	switch fe.Tag() {
	case "required", "required_with":
		return f + " is required"
	case "email":
		return f + " must be a valid email address"
	case "min":
		return fmt.Sprintf("%s must be at least %s%s", f, fe.Param(), unit)
	case "max":
		return fmt.Sprintf("%s must be at most %s%s", f, fe.Param(), unit)
	}
	return f + " is invalid"
}

// Ok sends a 200 {success: true} response. Takes the request context, the data and optional pagination meta; returns nothing.
func Ok(c *gin.Context, data any, meta *Meta) {
	c.JSON(http.StatusOK, Response{
		Success: true,
		Data:    data,
		Meta:    meta,
	})
}

// Created sends a 201 {success: true} response for a new resource. Takes the request context and the data; returns nothing.
func Created(c *gin.Context, data any) {
	c.JSON(http.StatusCreated, Response{
		Success: true,
		Data:    data,
	})
}

// ErrorHandler builds middleware that, after the handlers run, sends the last c.Error as the JSON error response.
// Takes nothing; returns the gin.HandlerFunc. Non-*AppError errors become a 500, so internal details never reach the client.
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
