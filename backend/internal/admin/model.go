package admin

import (
	"booking-system/pkg/response"
	"net/http"
)

var (
	// Shown to staff as is, so they're full sentences.
	errNoMovie     = response.NewError(http.StatusNotFound, "That film doesn't exist.")
	errNoEvent     = response.NewError(http.StatusNotFound, "That showing doesn't exist.")
	errBadRef      = response.NewError(http.StatusBadRequest, "That film, screen or format doesn't exist.")
	errPast        = response.NewError(http.StatusBadRequest, "Showings must start in the future.")
	errHasBookings = response.NewError(http.StatusConflict, "This showing has bookings, so it can't be changed or deleted.")
	errHasShowings = response.NewError(http.StatusConflict, "This film still has showings. Delete them first.")
	errOverlap     = response.NewError(http.StatusConflict, "That screen is showing another film then.")
	errTMDBIDInUse = response.NewError(http.StatusConflict, "Another film already has that TMDB ID.")
	errNoFormat    = response.NewError(http.StatusNotFound, "That format doesn't exist.")
	errFormatInUse = response.NewError(http.StatusConflict, "Showings use this format, so it can't be deleted.")
	errFormatName  = response.NewError(http.StatusConflict, "Another format already has that name.")
	errNoSeatType  = response.NewError(http.StatusNotFound, "That seat type doesn't exist.")
)
