package booking

import (
	"booking-system/pkg/middleware"
	"net/http"
)

// Routes mounts the booking endpoints under a showing: hold and release seats during checkout, then book them.
// All need a valid access token. Takes the mux; returns nothing.
func Routes(r *http.ServeMux) {
	r.HandleFunc("PUT /events/{id}/hold", middleware.Auth(Hold))
	r.HandleFunc("DELETE /events/{id}/hold", middleware.Auth(Release))
	r.HandleFunc("POST /events/{id}/bookings", middleware.Auth(Create))
}
