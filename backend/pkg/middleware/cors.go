package middleware

import (
	"log"
	"net/http"
	"os"
)

// CORS allows exactly one origin, with cookies: the whole value of ALLOWED_ORIGINS. It isn't split on commas.
// Wraps next; answers preflight OPTIONS requests itself with 204, and requests from any other origin with 403.
func CORS(next http.Handler) http.Handler {
	allowed := os.Getenv("ALLOWED_ORIGINS")
	if allowed == "" {
		log.Fatal("ALLOWED_ORIGINS is not set")
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if origin == "" || origin == "http://"+r.Host || origin == "https://"+r.Host {
			next.ServeHTTP(w, r) // same-origin, not a CORS request
			return
		}
		if origin != allowed {
			w.WriteHeader(http.StatusForbidden)
			return
		}

		h := w.Header()
		h.Set("Access-Control-Allow-Origin", origin)
		h.Set("Access-Control-Allow-Credentials", "true")
		h.Add("Vary", "Origin")
		if r.Method == http.MethodOptions {
			h.Set("Access-Control-Allow-Methods", "GET,POST,PATCH,DELETE")
			h.Set("Access-Control-Allow-Headers", "Origin,Content-Type,Authorization")
			h.Set("Access-Control-Max-Age", "43200") // 12 hours
			h.Add("Vary", "Access-Control-Request-Method")
			h.Add("Vary", "Access-Control-Request-Headers")
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.Set("Access-Control-Expose-Headers", "Content-Length")
		next.ServeHTTP(w, r)
	})
}
