package middleware

import (
	"log"
	"net"
	"net/http"
	"os"
	"time"
)

// Logger logs each request's status, duration, client IP, method and path once it has been handled.
// Wraps next; returns the wrapped handler.
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
		next.ServeHTTP(sw, r)
		log.Printf("%d | %v | %s | %s %q", sw.status, time.Since(start), ClientIP(r), r.Method, r.URL.Path)
	})
}

// statusWriter remembers the status code a handler wrote, for Logger.
type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(status int) {
	w.status = status
	w.ResponseWriter.WriteHeader(status)
}

// ClientIP returns the IP of the connecting peer. Proxy headers are ignored, since any client can forge them, unless
// TRUST_PROXY=true says every request comes through a proxy that sets X-Real-IP, like the frontend image's nginx.
// Takes the request; returns the IP, or "" if RemoteAddr can't be parsed.
func ClientIP(r *http.Request) string {
	if ip := r.Header.Get("X-Real-IP"); ip != "" && os.Getenv("TRUST_PROXY") == "true" {
		return ip
	}
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}
