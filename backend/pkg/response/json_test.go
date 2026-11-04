package response

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// TestBindTooLarge checks that a body over maxBody is refused with 413 rather than read in full.
// Takes the test runner; fails the test if Bind accepts it or returns another status.
func TestBindTooLarge(t *testing.T) {
	body := `{"firstName":"` + strings.Repeat("a", maxBody) + `"}`
	var s signup
	err := Bind(httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)), &s)
	if e, ok := errors.AsType[*AppError](err); !ok || e.Status != http.StatusRequestEntityTooLarge {
		t.Fatalf("got %v, want a 413 *AppError", err)
	}
}

type signup struct {
	FirstName string `json:"firstName" validate:"required"`
	Email     string `json:"email" validate:"required,email"`
	Password  string `json:"password" validate:"required,min=8"`
}

// TestBindMessage checks that each bad request body maps to the expected message.
// Takes the test runner; fails the test on any mismatch.
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
		err := decode(strings.NewReader(body), &s)
		if got := bindMessage(err); got != want {
			t.Errorf("%q: got %q, want %q", body, got, want)
		}
	}
}

// TestNewErrorCode checks that NewError derives the right Code from each status.
// Takes the test runner; fails the test on any mismatch.
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
