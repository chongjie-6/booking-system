// Mounts all the routes for the application
package router

import (
	"booking-system/internal/auth"
	"booking-system/internal/booking"
	"booking-system/internal/notification"
	"booking-system/internal/payment"
	"booking-system/internal/resource"
	"booking-system/internal/user"

	"github.com/gin-gonic/gin"
)

// New mounts every feature's routes onto the engine. Takes the Gin engine; returns nothing.
func New(r *gin.Engine) {
	user.Routes(r)
	booking.Routes(r)
	resource.Routes(r)
	payment.Routes(r)
	notification.Routes(r)
	auth.Routes(r)
}
