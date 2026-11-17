package booking

import (
	"booking-system/pkg/response"
	"net/http"
)

var (
	// Shown to the user as is, so they're full sentences.
	errNoShowing  = response.NewError(http.StatusNotFound, "That session has started or no longer exists.")
	errSeatsTaken = response.NewError(http.StatusConflict, "Some of those seats were just taken. Please choose others.")
)
