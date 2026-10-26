package user

import (
	"booking-system/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// Routes mounts the /user endpoints; GET and PATCH need a valid access token. Takes the Gin engine; returns nothing.
func Routes(r *gin.Engine) {
	userRoutes := r.Group("/user")

	userRoutes.GET("/ping", Ping)

	userRoutes.GET("", middleware.Auth(), GetUser)
	userRoutes.PATCH("", middleware.Auth(), PatchUser)
}
