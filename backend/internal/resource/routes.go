package resource

import (
	"booking-system/pkg/response"
	"net/http"
)

// Routes mounts the public catalog endpoints: movies with their showings, and each showing's seats. Takes the mux; returns nothing.
func Routes(r *http.ServeMux) {
	r.HandleFunc("GET /resources/ping", func(w http.ResponseWriter, _ *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"message": "pong"})
	})

	r.HandleFunc("GET /movies", ListMovies)
	r.HandleFunc("GET /events/{id}/seats", ListSeats)
}
