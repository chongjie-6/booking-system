package user

import (
	"booking-system/pkg/middleware"
	"net/http"
)

// Routes mounts the /user endpoints; GET and PATCH need a valid access token. Takes the mux; returns nothing.
func Routes(r *http.ServeMux) {
	r.HandleFunc("GET /user/ping", Ping)

	r.HandleFunc("GET /user", middleware.Auth(GetUser))
	r.HandleFunc("PATCH /user", middleware.Auth(PatchUser))
}
