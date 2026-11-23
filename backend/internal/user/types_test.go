package user

import (
	"booking-system/pkg/response"
	"net/http/httptest"
	"strings"
	"testing"
)

// PATCH /user leaves out fields that aren't sent, but a field that is sent must still be valid.
func TestPatchUserRequestValidation(t *testing.T) {
	cases := []struct {
		body string
		ok   bool
	}{
		{`{}`, true},
		{`{"firstName": "Ada"}`, true},
		{`{"firstName": ""}`, false},
		{`{"lastName": ""}`, false},
		{`{"email": ""}`, false},
		{`{"email": "not-an-email"}`, false},
		{`{"email": "ada@example.com"}`, true},
	}
	for _, c := range cases {
		r := httptest.NewRequest("PATCH", "/user", strings.NewReader(c.body))
		var req PatchUserRequest
		if err := response.Bind(r, &req); (err == nil) != c.ok {
			t.Errorf("Bind(%s) = %v, want ok %v", c.body, err, c.ok)
		}
	}
}
