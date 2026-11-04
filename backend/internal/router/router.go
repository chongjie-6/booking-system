// Mounts all the routes for the application
package router

import (
	"booking-system/internal/auth"
	"booking-system/internal/booking"
	"booking-system/internal/notification"
	"booking-system/internal/payment"
	"booking-system/internal/resource"
	"booking-system/internal/user"
	"net/http"
)

// New mounts every feature's routes onto the mux. Takes the mux; returns nothing.
func New(r *http.ServeMux) {
	user.Routes(r)
	booking.Routes(r)
	resource.Routes(r)
	payment.Routes(r)
	notification.Routes(r)
	auth.Routes(r)
}
