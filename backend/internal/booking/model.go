package booking

import (
	"booking-system/pkg/response"
	"net/http"
)

var (
	// Shown to the user as is, so they're full sentences.
	errNoShowing    = response.NewError(http.StatusNotFound, "That session has started or no longer exists.")
	errSeatsTaken   = response.NewError(http.StatusConflict, "Some of those seats were just taken. Please choose others.")
	errPriceChanged = response.NewError(http.StatusConflict, "The price of this session has changed. Please start again.")
	errNoPayments   = response.NewError(http.StatusServiceUnavailable, "Payments aren't available right now. Please try again later.")
	errNoPayment    = response.NewError(http.StatusNotFound, "We couldn't find that payment.")
	errNotPaid      = response.NewError(http.StatusPaymentRequired, "Your payment hasn't gone through yet.")
	errRefunded     = response.NewError(http.StatusConflict, "This payment has been refunded.")
	errNoBooking    = response.NewError(http.StatusNotFound, "We couldn't find that booking.")
	errTooLate      = response.NewError(http.StatusConflict, "Bookings can't be cancelled within an hour of the showing.")
	errRefundFailed = response.NewError(http.StatusBadGateway, "We couldn't refund your payment just now. Please try again.")
)
