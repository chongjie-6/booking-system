package main

import (
	"booking-system/pkg/middleware"
	"booking-system/pkg/redis"
	"booking-system/pkg/response"
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	// Load .env variables into process
	godotenv.Load()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rdb := redis.New(ctx)
	defer rdb.Close()

	// Create a new Gin router with default middleware
	r := gin.Default()
	r.SetTrustedProxies(nil)

	// Apply middleware & security headers
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS())
	r.Use(response.ErrorHandler())
	r.Use(middleware.RateLimiterMiddleware(middleware.New(rdb), func(c *gin.Context) string {
		return c.ClientIP()
	}))

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
