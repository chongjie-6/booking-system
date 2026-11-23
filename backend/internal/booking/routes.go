package booking

import (
	"booking-system/pkg/middleware"
	"net/http"
)

// Routes mounts the booking endpoints: hold and release seats during checkout, pay for and book them, and see or cancel
// the user's bookings. All but Stripe's webhook need a valid access token. Takes the mux; returns nothing.
func Routes(r *http.ServeMux) {
	r.HandleFunc("PUT /events/{id}/hold", middleware.Auth(Hold))
	r.HandleFunc("DELETE /events/{id}/hold", middleware.Auth(Release))
	r.HandleFunc("POST /events/{id}/payment", middleware.Auth(Pay))
	r.HandleFunc("POST /events/{id}/bookings", middleware.Auth(Create))
	r.HandleFunc("GET /bookings", middleware.Auth(List))
	r.HandleFunc("GET /bookings/{ref}", middleware.Auth(Get))
	r.HandleFunc("DELETE /bookings/{id}", middleware.Auth(Cancel))
	r.HandleFunc("POST /payments/webhook", Webhook)
}
