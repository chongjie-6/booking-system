package main

import (
	"booking-system/internal/app"
	"booking-system/internal/router"
	"booking-system/pkg/response"
	"context"
	"net/http"

	"github.com/joho/godotenv"
)

// main loads .env, connects external dependencies, builds the server with middleware and routes, then serves it.
// Takes no input and returns nothing; it blocks until Ctrl+C or SIGTERM, then shuts down gracefully.
func main() {
	// Load .env variables into process
	godotenv.Load()
	ctx := context.Background()

	// Connect to postgres + redis + kafka + email and open the GeoIP database
	app.Connect(ctx)

	r := http.NewServeMux()

	// Return JSON on not found routes; "/" matches whatever no other route does
	r.HandleFunc("/", func(w http.ResponseWriter, _ *http.Request) {
		response.Error(w, response.ErrNotFound)
	})

	// Health check route
	r.HandleFunc("GET /health", func(w http.ResponseWriter, _ *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"message": "OK"})
	})

	// Mount routes
	router.New(r)

	// Apply middleware & security headers, then serve until Ctrl+C or SIGTERM and shut down gracefully
	app.Serve(ctx, app.Middleware(r))
}
