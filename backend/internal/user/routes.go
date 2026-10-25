package user

import (
	"booking-system/pkg/middleware"

	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	userRoutes := r.Group("/user")

	userRoutes.GET("/ping", Ping)

	userRoutes.POST("/refresh", RefreshToken)
	userRoutes.POST("", CreateUser)
	userRoutes.GET("", middleware.Auth(), GetUser)
	userRoutes.PATCH("", middleware.Auth(), PatchUser)
}
