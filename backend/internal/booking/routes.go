package booking

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Routes mounts the /bookings endpoints (just a ping for now). Takes the Gin engine; returns nothing.
func Routes(r *gin.Engine) {
	bookingRoutes := r.Group("/bookings")
	{
		bookingRoutes.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "pong"}) })
	}
}
