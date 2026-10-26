package notification

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Routes mounts the /notifications endpoints (just a ping for now). Takes the Gin engine; returns nothing.
func Routes(r *gin.Engine) {
	notificationRoutes := r.Group("/notifications")
	{
		notificationRoutes.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "pong"}) })
	}
}
