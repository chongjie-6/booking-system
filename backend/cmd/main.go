package main

import (
	"booking-system/internal/router"
	"booking-system/pkg/db"
	"booking-system/pkg/geoip"
	"booking-system/pkg/middleware"
	"booking-system/pkg/mq"
	"booking-system/pkg/rdb"
	"booking-system/pkg/response"
	"cmp"
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

// main loads .env, connects Postgres, Redis and Kafka, runs migrations, then starts the Gin server with all middleware and routes.
// Takes no input and returns nothing; it blocks until Ctrl+C or SIGTERM, then shuts down gracefully.
func main() {
	// Load .env variables into process
	godotenv.Load()
	gin.SetMode(os.Getenv("GIN_MODE"))
	ctx := context.Background()

	// Connect to redis + database + mq
	db.Connect(ctx)
	db.Migrate()
	rdb.Connect(ctx)
	geoip.Open()
	mq.Connect()

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

	// Serve until Ctrl+C or SIGTERM, then let in-flight requests finish before sending queued Kafka messages.
	sig, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	srv := &http.Server{Addr: ":" + cmp.Or(os.Getenv("PORT"), "8080"), Handler: r} // same port r.Run() picked
	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()
	<-sig.Done()

	shutdownCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	srv.Shutdown(shutdownCtx) // waits for in-flight logins, which may still queue messages
	if mq.Client != nil {
		mq.Client.Flush(shutdownCtx)
		mq.Client.Close()
	}
}
