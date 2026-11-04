package resource

import (
	"booking-system/pkg/response"
	"net/http"
)

// Routes mounts the /resources endpoints (just a ping for now). Takes the mux; returns nothing.
func Routes(r *http.ServeMux) {
	r.HandleFunc("GET /resources/ping", func(w http.ResponseWriter, _ *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"message": "pong"})
	})
}
