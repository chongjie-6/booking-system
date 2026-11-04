package auth

import (
	"booking-system/pkg/middleware"
	"booking-system/pkg/rdb"
	"net/http"
	"time"
)

// Routes mounts the /auth endpoints (ping, refresh, login, logout, register, password change and reset).
// Takes the mux; returns nothing. Call after rdb.Connect, since the stricter limiter needs its client.
func Routes(r *http.ServeMux) {
	// Login and reset emails also get 5 tries per IP, refilled one a minute: enough for typos, too slow for password
	// guessing or flooding someone's inbox. The key prefix gives each route its own bucket, apart from the app-wide one.
	strict := middleware.NewTokenBucket(middleware.TokenBucketConfig{Client: rdb.Client, Capacity: 5, RefillRate: 1, RefillInterval: time.Minute})
	limit := func(name string, h http.HandlerFunc) http.Handler {
		return middleware.RateLimiterMiddleware(strict, func(r *http.Request) string { return name + ":" + middleware.ClientIP(r) }, h)
	}

	r.HandleFunc("GET /auth/ping", Ping)

	r.HandleFunc("POST /auth/refresh", RefreshToken)
	r.Handle("POST /auth/login", limit("login", Login))
	r.HandleFunc("POST /auth/logout", Logout)
	r.HandleFunc("POST /auth/register", Register)
	r.HandleFunc("POST /auth/password", middleware.Auth(ChangePassword))
	r.Handle("POST /auth/password/forgot", limit("forgot", ForgotPassword))
	r.HandleFunc("POST /auth/password/reset", ResetPassword)
}
