package auth

import (
	"booking-system/pkg/middleware"

	"github.com/gin-gonic/gin"
)

// Routes mounts the /auth endpoints (ping, refresh, login, logout, register, password). Takes the Gin engine; returns nothing.
func Routes(r *gin.Engine) {
	authRoutes := r.Group("/auth")

	authRoutes.GET("/ping", Ping)

	authRoutes.POST("/refresh", RefreshToken)
	authRoutes.POST("/login", Login)
	authRoutes.POST("/logout", Logout)
	authRoutes.POST("/register", Register)
	authRoutes.POST("/password", middleware.Auth(), ChangePassword)
}
