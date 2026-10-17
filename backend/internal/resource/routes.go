package resource

import (
	"github.com/gin-gonic/gin"
)

func RegisterResourceRoutes(r *gin.Engine) {
	resourceRoutes := r.Group("/resources")
	{
		resourceRoutes.GET("/ping")
	}
}
