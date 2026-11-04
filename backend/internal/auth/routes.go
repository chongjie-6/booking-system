package auth

import (
	"booking-system/pkg/middleware"
	"net/http"
)

// Routes mounts the /auth endpoints (ping, refresh, login, logout, register, password). Takes the mux; returns nothing.
func Routes(r *http.ServeMux) {
	r.HandleFunc("GET /auth/ping", Ping)

	r.HandleFunc("POST /auth/refresh", RefreshToken)
	r.HandleFunc("POST /auth/login", Login)
	r.HandleFunc("POST /auth/logout", Logout)
	r.HandleFunc("POST /auth/register", Register)
	r.HandleFunc("POST /auth/password", middleware.Auth(ChangePassword))
}
