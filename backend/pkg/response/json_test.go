package response

import (
	"testing"

	"github.com/gin-gonic/gin/binding"
)

type signup struct {
	FirstName string `json:"firstName" binding:"required"`
	Email     string `json:"email" binding:"required,email"`
	Password  string `json:"password" binding:"required,min=8"`
}

func TestBindMessage(t *testing.T) {
	cases := map[string]string{
		``:                                 "request body is empty",
		`{"firstName":`:                    "request body is not valid JSON",
		`{nope}`:                           "request body is not valid JSON",
		`[]`:                               "request body must be a JSON object",
		`{"firstName":1}`:                  "firstName must be a string",
		`{"email":"x","password":"short"}`: "firstName is required; email must be a valid email address; password must be at least 8 characters",
	}
	for body, want := range cases {
		var s signup
		err := binding.JSON.BindBody([]byte(body), &s)
		if got := bindMessage(err); got != want {
			t.Errorf("%q: got %q, want %q", body, got, want)
		}
	}
}

func TestNewErrorCode(t *testing.T) {
	for err, want := range map[*AppError]string{
		ErrBadRequest:          "BAD_REQUEST",
		ErrConflict:            "CONFLICT",
		ErrTooManyRequests:     "TOO_MANY_REQUESTS",
		ErrInternalServerError: "INTERNAL_SERVER_ERROR",
	} {
		if err.Code != want {
			t.Errorf("got %q, want %q", err.Code, want)
		}
	}
}
