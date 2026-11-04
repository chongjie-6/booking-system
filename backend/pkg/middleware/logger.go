package middleware

import (
	"log"
	"net"
	"net/http"
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

// ClientIP returns the IP of the connecting peer. Proxy headers like X-Forwarded-For are ignored, since any client can forge them.
// Takes the request; returns the IP, or "" if RemoteAddr can't be parsed.
func ClientIP(r *http.Request) string {
	ip, _, _ := net.SplitHostPort(r.RemoteAddr)
	return ip
}
