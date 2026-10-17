package notification

import (
	"github.com/gin-gonic/gin"
)

func RegisterNotificationRoutes(r *gin.Engine){
	notificationRoutes := r.Group("/notifications")
	{
		notificationRoutes.GET("/ping")
	}
}