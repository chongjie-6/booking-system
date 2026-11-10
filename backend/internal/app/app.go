// Connects external dependencies, applies global middleware, and runs the server with graceful shutdown
package app

import (
	"booking-system/internal/notification"
	"booking-system/internal/outbox"
	"booking-system/pkg/db"
	"booking-system/pkg/email"
	"booking-system/pkg/geoip"
	"booking-system/pkg/middleware"
	"booking-system/pkg/mq"
	"booking-system/pkg/rdb"
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

// Middleware wraps h in request logging, security headers, CORS and the per-IP rate limiter, outermost first.
// Takes the handler; returns the wrapped handler.
func Middleware(h http.Handler) http.Handler {
	h = middleware.RateLimiterMiddleware(middleware.NewBucket(), middleware.ClientIP, h)
	h = middleware.CORS(h)
	h = middleware.SecurityHeaders(h)
	return middleware.Logger(h)
}

// Serve starts the outbox pruner and notification consumer and serves r until Ctrl+C or SIGTERM, then shuts down gracefully.
// Outbox events are sent to Kafka by Debezium, not by this process. Takes a parent context and the handler; returns once shutdown has finished.
func Serve(ctx context.Context, h http.Handler) {
	sig, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	// Consumers stop on sig; shutdown waits for them so their last batch is handled and committed before exiting.
	var consumers sync.WaitGroup
	consumers.Go(func() { outbox.Prune(sig) })
	consumers.Go(func() { notification.Consume(sig) })

	srv := &http.Server{Addr: ":" + cmp.Or(os.Getenv("PORT"), "8080"), Handler: h}
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
