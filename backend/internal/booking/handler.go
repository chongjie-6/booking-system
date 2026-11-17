package booking

import (
	"booking-system/internal/notification"
	"booking-system/pkg/db"
	"booking-system/pkg/middleware"
	"booking-system/pkg/response"
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// Hold holds the chosen seats for the signed-in user while they check out. Takes the request with the event ID in the path
// and a seatsRequest body; responds 200, or 400 bad body, 404 no upcoming showing, 409 seat booked or held by someone else, 500.
func Hold(w http.ResponseWriter, r *http.Request) {
	var req seatsRequest
	if err := response.Bind(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	eventID, userID := r.PathValue("id"), middleware.User(r).UserID
	if _, err := checkSeats(r.Context(), db.Pool, eventID, req.SeatIDs); err != nil {
		response.Error(w, err)
		return
	}
	if err := hold(r.Context(), eventID, userID, req.SeatIDs); err != nil {
		response.Error(w, err)
		return
	}
	response.Ok(w, nil, nil)
}

// Release drops the signed-in user's holds on the seats, e.g. when they go back to change seats.
// Takes the request with the event ID in the path and a seatsRequest body; responds 200, or 400 bad body.
func Release(w http.ResponseWriter, r *http.Request) {
	var req seatsRequest
	if err := response.Bind(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	release(r.Context(), r.PathValue("id"), middleware.User(r).UserID, req.SeatIDs)
	response.Ok(w, nil, nil)
}

// Create books the seats for the signed-in user and queues the confirmation email in the same transaction.
// Takes the request with the event ID in the path and a seatsRequest body; responds 201 with the Booking,
// or 400 bad body, 404 no upcoming showing, 409 seat booked or held by someone else, 500.
// ponytail: no payment provider yet, so a booking is confirmed without charging; take payment between the hold and the insert.
func Create(w http.ResponseWriter, r *http.Request) {
	var req seatsRequest
	if err := response.Bind(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	ctx, eventID, userID := r.Context(), r.PathValue("id"), middleware.User(r).UserID

	// Refreshes the user's own hold, and stops them taking seats someone else is checking out with.
	if err := hold(ctx, eventID, userID, req.SeatIDs); err != nil {
		response.Error(w, err)
		return
	}

	var b Booking
	err := pgx.BeginFunc(ctx, db.Pool, func(tx pgx.Tx) error {
		roomID, err := checkSeats(ctx, tx, eventID, req.SeatIDs)
		if err != nil {
			return err
		}
		if b.BookingID, err = dbCreateBooking(ctx, tx, eventID, roomID, userID, req.SeatIDs); err != nil {
			return err
		}
		// The UUIDv7's last 8 hex digits are random, so they make a short reference that's easy to read out.
		// bookings_event_ref_idx keeps it unique within a showing.
		b.Ref = strings.ToUpper(b.BookingID[len(b.BookingID)-8:])
		return notification.SendBookingConfirmedNotification(ctx, tx, userID, b.BookingID, b.Ref)
	})
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" { // a seat sold since checkSeats
		err = errSeatsTaken
	}
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Created(w, b)
}

// checkSeats makes sure the showing hasn't started and every seat is in its room and unbooked. Takes a context, querier,
// event ID and seats; returns the showing's room ID, errNoShowing, errSeatsTaken, or the database error.
func checkSeats(ctx context.Context, q db.Querier, eventID string, seatIDs []string) (string, error) {
	f, err := dbFreeSeats(ctx, q, eventID, seatIDs)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); errors.Is(err, pgx.ErrNoRows) || ok && pgErr.Code == "22P02" { // 22P02: not a UUID
		return "", errNoShowing
	}
	if err != nil {
		return "", err
	}
	if f.Free < len(seatIDs) {
		return "", errSeatsTaken
	}
	return f.RoomID, nil
}
