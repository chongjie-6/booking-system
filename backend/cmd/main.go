package main

import (
	"booking-system/internal/app"
	"booking-system/internal/router"
	"booking-system/pkg/response"
	"context"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// main loads .env, connects external dependencies, builds the Gin server with middleware and routes, then serves it.
// Takes no input and returns nothing; it blocks until Ctrl+C or SIGTERM, then shuts down gracefully.
func main() {
	// Load .env variables into process
	godotenv.Load()
	gin.SetMode(os.Getenv("GIN_MODE"))
	ctx := context.Background()

	// Connect to postgres + redis + kafka + email and open the GeoIP database
	app.Connect(ctx)

	// Create a new Gin router with default middleware, trusting no proxy headers for the client IP
	r := gin.Default()
	r.SetTrustedProxies(nil)

	// Apply middleware & security headers
	app.Middleware(r)

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

	// Mount routes
	router.New(r)

	// Serve until Ctrl+C or SIGTERM, then shut down gracefully
	app.Serve(ctx, r)
}
