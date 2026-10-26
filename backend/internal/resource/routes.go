package resource

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Routes mounts the /resources endpoints (just a ping for now). Takes the Gin engine; returns nothing.
func Routes(r *gin.Engine) {
	resourceRoutes := r.Group("/resources")
	{
		resourceRoutes.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "pong"}) })
	}
}
