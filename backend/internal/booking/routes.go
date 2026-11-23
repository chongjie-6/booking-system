package booking

import (
	"booking-system/pkg/middleware"
	"net/http"
)

// Routes mounts the booking endpoints: hold and release seats during checkout, book them, and list the user's bookings.
// All need a valid access token. Takes the mux; returns nothing.
func Routes(r *http.ServeMux) {
	r.HandleFunc("PUT /events/{id}/hold", middleware.Auth(Hold))
	r.HandleFunc("DELETE /events/{id}/hold", middleware.Auth(Release))
	r.HandleFunc("POST /events/{id}/bookings", middleware.Auth(Create))
	r.HandleFunc("GET /bookings", middleware.Auth(List))
}
