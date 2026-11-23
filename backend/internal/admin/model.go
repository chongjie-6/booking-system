package admin

import (
	"booking-system/pkg/response"
	"net/http"
)

var (
	// Shown to staff as is, so they're full sentences.
	errNoMovie     = response.NewError(http.StatusNotFound, "That film doesn't exist.")
	errNoEvent     = response.NewError(http.StatusNotFound, "That showing doesn't exist.")
	errBadRef      = response.NewError(http.StatusBadRequest, "That film or screen doesn't exist.")
	errPast        = response.NewError(http.StatusBadRequest, "Showings must start in the future.")
	errHasBookings = response.NewError(http.StatusConflict, "This showing has bookings, so it can't be changed or deleted.")
	errHasShowings = response.NewError(http.StatusConflict, "This film still has showings. Delete them first.")
	errOverlap     = response.NewError(http.StatusConflict, "That screen is showing another film then.")
	errTMDBIDInUse = response.NewError(http.StatusConflict, "Another film already has that TMDB ID.")
)
