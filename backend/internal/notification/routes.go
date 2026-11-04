package notification

import (
	"booking-system/pkg/response"
	"net/http"
)

// Routes mounts the /notifications endpoints (just a ping for now). Takes the mux; returns nothing.
func Routes(r *http.ServeMux) {
	r.HandleFunc("GET /notifications/ping", func(w http.ResponseWriter, _ *http.Request) {
		response.JSON(w, http.StatusOK, map[string]string{"message": "pong"})
	})
}
