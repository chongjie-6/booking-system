// Connects external dependencies, applies global middleware, and runs the server with graceful shutdown
package app

import (
	"booking-system/internal/notification"
	"booking-system/pkg/db"
	"booking-system/pkg/email"
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
	"sync"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// Connect connects Postgres (and runs migrations), Redis, GeoIP, Kafka and Resend. Takes a context for the connections; returns nothing.
func Connect(ctx context.Context) {
	db.Connect(ctx)
	db.Migrate()
	rdb.Connect(ctx)
	geoip.Open()
	mq.Connect()
	email.Connect()
}

// Middleware applies security headers, CORS, the JSON error handler and the per-IP rate limiter. Takes the Gin engine; returns nothing.
func Middleware(r *gin.Engine) {
	r.Use(middleware.SecurityHeaders())
	r.Use(middleware.CORS())
	r.Use(response.ErrorHandler())
	r.Use(middleware.RateLimiterMiddleware(middleware.NewBucket(), func(c *gin.Context) string {
		return c.ClientIP()
	}))
}

// Serve starts the notification consumer and serves r until Ctrl+C or SIGTERM, then shuts down gracefully.
// Takes a parent context and the engine; returns once shutdown has finished.
func Serve(ctx context.Context, r *gin.Engine) {
	sig, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Consumers stop on sig; shutdown waits for them so their last batch is handled and committed before exiting.
	var consumers sync.WaitGroup
	consumers.Go(func() { notification.Consume(sig) })

	srv := &http.Server{Addr: ":" + cmp.Or(os.Getenv("PORT"), "8080"), Handler: r} // same port r.Run() picked
	go func() {
		if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("server: %v", err)
		}
	}()
	<-sig.Done()

	shutdown(ctx, srv, &consumers)
}

// shutdown lets in-flight requests finish, waits for consumers, then sends queued Kafka messages and closes the producer.
// Takes a parent context, the server and the consumers' WaitGroup; returns nothing. Gives up on requests and the flush after 10s.
func shutdown(ctx context.Context, srv *http.Server, consumers *sync.WaitGroup) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	srv.Shutdown(ctx) // waits for in-flight logins, which may still queue messages
	consumers.Wait()  // before closing the producer, since handlers may produce events too
	if mq.Client != nil {
		mq.Client.Flush(ctx)
		mq.Client.Close()
	}
}
