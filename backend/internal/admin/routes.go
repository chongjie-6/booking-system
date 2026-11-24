package admin

import (
	"booking-system/pkg/middleware"
	"net/http"
)

// Routes mounts the staff endpoints for managing films, showings, formats and seat surcharges. All need an admin's access token.
// Screens are set up with SQL (see backend/seed.sql). Takes the mux; returns nothing.
func Routes(r *http.ServeMux) {
	r.HandleFunc("GET /admin/movies", middleware.Admin(ListMovies))
	r.HandleFunc("POST /admin/movies", middleware.Admin(CreateMovie))
	r.HandleFunc("PUT /admin/movies/{id}", middleware.Admin(UpdateMovie))
	r.HandleFunc("DELETE /admin/movies/{id}", middleware.Admin(DeleteMovie))
	r.HandleFunc("GET /admin/rooms", middleware.Admin(ListRooms))
	r.HandleFunc("GET /admin/formats", middleware.Admin(ListFormats))
	r.HandleFunc("POST /admin/formats", middleware.Admin(CreateFormat))
	r.HandleFunc("PUT /admin/formats/{id}", middleware.Admin(UpdateFormat))
	r.HandleFunc("DELETE /admin/formats/{id}", middleware.Admin(DeleteFormat))
	r.HandleFunc("GET /admin/surcharges", middleware.Admin(ListSurcharges))
	r.HandleFunc("PUT /admin/surcharges/{type}", middleware.Admin(UpdateSurcharge))
	r.HandleFunc("POST /admin/events", middleware.Admin(CreateEvent))
	r.HandleFunc("PUT /admin/events/{id}", middleware.Admin(UpdateEvent))
	r.HandleFunc("DELETE /admin/events/{id}", middleware.Admin(DeleteEvent))
}
