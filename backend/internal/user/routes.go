package user

import (
	"github.com/gin-gonic/gin"
)

func RegisterUserRoutes(r *gin.Engine){
	userRoutes := r.Group("/users")
	{
		userRoutes.GET("/ping")
	}
}