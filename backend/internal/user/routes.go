package user

import (
	"github.com/gin-gonic/gin"
)

func Routes(r *gin.Engine) {
	userRoutes := r.Group("/user")

	userRoutes.GET("/ping", Ping)

	userRoutes.POST("", CreateUser)
}
