package response

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"reflect"
	"strings"

	"github.com/go-playground/validator/v10"
)

// validate checks request structs against their `validate` tags. Errors use the JSON field name (firstName) instead of the Go one (FirstName).
var validate = func() *validator.Validate {
	v := validator.New()
	v.RegisterTagNameFunc(func(f reflect.StructField) string {
		return strings.SplitN(f.Tag.Get("json"), ",", 2)[0]
	})
	return v
}()

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

// maxBody caps how much of a request body Bind reads, so a huge body can't tie up memory. Request bodies here are small JSON.
const maxBody = 1 << 20 // 1 MiB

// Bind decodes the JSON request body into v and validates it. Takes the request and a pointer to the struct;
// returns nil, a 413 *AppError if the body is over maxBody, or a 400 *AppError that tells the user what to fix.
func Bind(r *http.Request, v any) error {
	if err := decode(http.MaxBytesReader(nil, r.Body, maxBody), v); err != nil {
		if _, ok := errors.AsType[*http.MaxBytesError](err); ok {
			return NewError(http.StatusRequestEntityTooLarge, "request body is too large")
		}
		return NewError(http.StatusBadRequest, bindMessage(err))
	}
	return nil
}

// decode reads JSON from body into v, then validates v. Takes the body and a pointer to the struct; returns the decode or validation error.
func decode(body io.Reader, v any) error {
	if err := json.NewDecoder(body).Decode(v); err != nil {
		return err
	}
	return validate.Struct(v)
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

// JSON sends v as a JSON response with the given status. Takes the writer, status and value; returns nothing.
func JSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("write response: %v", err)
	}
}

// Ok sends a 200 {success: true} response. Takes the writer, the data and optional pagination meta; returns nothing.
func Ok(w http.ResponseWriter, data any, meta *Meta) {
	JSON(w, http.StatusOK, Response{Success: true, Data: data, Meta: meta})
}

// Created sends a 201 {success: true} response for a new resource. Takes the writer and the data; returns nothing.
func Created(w http.ResponseWriter, data any) {
	JSON(w, http.StatusCreated, Response{Success: true, Data: data})
}

// Error sends err as the JSON error response. Takes the writer and the error; returns nothing.
// Non-*AppError errors become a 500, so internal details never reach the client.
func Error(w http.ResponseWriter, err error) {
	appError, ok := errors.AsType[*AppError](err)
	if !ok {
		appError = ErrInternalServerError
	}
	JSON(w, appError.Status, &Response{Success: false, Error: appError})
}
