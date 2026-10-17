package main

import (
	"booking-system/pkg/middleware"
	"booking-system/pkg/response"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env variables into process
	godotenv.Load()

	r := gin.Default()
	r.SetTrustedProxies(nil)

	r.Use(middleware.RateLimiter())

	r.Use(middleware.CORS())

	r.Use(response.ErrorHandler())

	// Return JSON on not found routes
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
