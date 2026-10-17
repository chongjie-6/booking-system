package main

import (
	"booking-system/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
)

func main() {
	// Create a new Gin router with default middleware
	r := gin.Default()

	// Use error handler as middleware
	r.Use(response.ErrorHandler())

	r.NoRoute(func(c *gin.Context) {
		c.Error(response.ErrNotFound)
	})

	// Health check route mounted before any other routes
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "OK",
		})
	})

	// Start server on port 8080
	r.Run()
}
