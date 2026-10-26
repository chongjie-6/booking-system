package main

import (
	"booking-system/internal/router"
	"booking-system/pkg/db"
	"booking-system/pkg/middleware"
	"booking-system/pkg/rdb"
	"booking-system/pkg/response"
	"context"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// main loads .env, connects Postgres and Redis, runs migrations, then starts the Gin server with all middleware and routes.
// Takes no input and returns nothing; it blocks while the server runs.
func main() {
	// Load .env variables into process
	godotenv.Load()
	gin.SetMode(os.Getenv("GIN_MODE"))
	ctx := context.Background()

	// Connect to redis + database
	db.Connect(ctx)
	db.Migrate()
	rdb.Connect(ctx)

	// Create a new Gin router with default middleware
	r := gin.Default()
	r.SetTrustedProxies(nil)

	// Apply middleware & security headers
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS())
	r.Use(response.ErrorHandler())
	r.Use(middleware.RateLimiterMiddleware(middleware.NewBucket(), func(c *gin.Context) string {
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

	router.New(r)

	// Start server on port 8080
	r.Run()
}
