package booking

import (
	"github.com/gin-gonic/gin"
)

func RegisterBookingRoutes(r *gin.Engine){
	bookingRoutes := r.Group("/bookings")
	{
		bookingRoutes.GET("/ping")
	}
}