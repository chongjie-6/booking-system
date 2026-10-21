package user

import (
	"booking-system/pkg/middleware"
	"booking-system/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	userRoutes := r.Group("/users")
	{
		userRoutes.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "pong"}) })
		userRoutes.GET("/me", middleware.Auth(), func(c *gin.Context) { response.Ok(c, c.MustGet("user"), nil) })
	}
}
