package payment

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// Routes mounts the /payments endpoints (just a ping for now). Takes the Gin engine; returns nothing.
func Routes(r *gin.Engine) {
	paymentRoutes := r.Group("/payments")
	{
		paymentRoutes.GET("/ping", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{"message": "pong"}) })
	}
}
