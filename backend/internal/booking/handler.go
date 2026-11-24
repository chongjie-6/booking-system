package booking

import (
	"booking-system/internal/notification"
	"booking-system/pkg/db"
	"booking-system/pkg/middleware"
	"booking-system/pkg/response"
	"booking-system/pkg/stripe"
	"context"
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// cancelWindow is how close to the showing a booking can still be cancelled and refunded.
const cancelWindow = time.Hour

// Hold holds the chosen seats for the signed-in user while they check out. Takes the request with the event ID in the path
// and a seatsRequest body; responds 200 with a holdResponse, or 400 bad body, 404 no upcoming showing,
// 409 seat booked or held by someone else, 500.
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
	response.Ok(w, holdResponse{int(holdTTL.Seconds())}, nil)
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

// Pay refreshes the user's hold on the seats and starts a Stripe payment for them. The seats go in the payment's metadata,
// so the booking made from it is for exactly what was paid for. Takes the request with the event ID in the path and a
// seatsRequest body; responds 200 with a paymentResponse, or 400 bad body, 404 no upcoming showing,
// 409 seat booked or held by someone else, 503 Stripe not set up or unreachable.
func Pay(w http.ResponseWriter, r *http.Request) {
	if !stripe.Enabled() {
		response.Error(w, errNoPayments)
		return
	}
	var req seatsRequest
	if err := response.Bind(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	ctx, eventID, userID := r.Context(), r.PathValue("id"), middleware.User(r).UserID
	if err := hold(ctx, eventID, userID, req.SeatIDs); err != nil {
		response.Error(w, err)
		return
	}
	f, err := checkSeats(ctx, db.Pool, eventID, req.SeatIDs)
	if err != nil {
		response.Error(w, err)
		return
	}
	amount := f.TotalCents
	pi, err := stripe.CreatePaymentIntent(ctx, amount, map[string]string{
		"event_id": eventID, "user_id": userID, "seat_ids": strings.Join(req.SeatIDs, ","), // 10 UUIDs fit Stripe's 500 characters
	})
	if err != nil {
		log.Printf("create payment: %v", err)
		response.Error(w, errNoPayments)
		return
	}
	response.Ok(w, paymentResponse{pi.ClientSecret, os.Getenv("STRIPE_PUBLISHABLE_KEY"), amount}, nil)
}

// Create books the seats a succeeded payment paid for, called by the checkout page once Stripe has taken the payment.
// Takes the request with the event ID in the path and a bookRequest body; responds 201 with the Booking, or 400 bad body,
// 402 not paid yet, 404 no such payment for this user and showing, 409 seats or showing gone (and refunded), 500.
func Create(w http.ResponseWriter, r *http.Request) {
	var req bookRequest
	if err := response.Bind(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	b, err := confirm(r.Context(), req.PaymentIntentID, r.PathValue("id"), middleware.User(r).UserID)
	if err != nil {
		log.Printf("book %s: %v", req.PaymentIntentID, err)
		response.Error(w, err)
		return
	}
	response.Created(w, b)
}

// Webhook books the seats for each payment_intent.succeeded event Stripe sends, so a booking is made even if the buyer
// closes the tab straight after paying. Takes Stripe's request; responds 200 once the event is handled (booked, already
// booked, or refunded), 400 if the signature is wrong, or 500 so that Stripe sends it again later.
func Webhook(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 64<<10))
	if err == nil {
		err = stripe.VerifyWebhook(body, r.Header.Get("Stripe-Signature"), os.Getenv("STRIPE_WEBHOOK_SECRET"), time.Now())
	}
	var e struct {
		Type string `json:"type"`
		Data struct {
			Object struct {
				ID string `json:"id"`
			} `json:"object"`
		} `json:"data"`
	}
	if err == nil {
		err = json.Unmarshal(body, &e)
	}
	if err != nil {
		response.Error(w, response.ErrBadRequest)
		return
	}
	if e.Type == "payment_intent.succeeded" {
		_, err := confirm(r.Context(), e.Data.Object.ID, "", "")
		if _, settled := errors.AsType[*response.AppError](err); err != nil && !settled {
			log.Printf("webhook %s: %v", e.Data.Object.ID, err)
			response.Error(w, err)
			return
		}
	}
	response.Ok(w, nil, nil)
}

// List returns the signed-in user's bookings, latest showing first. Takes the request; responds 200 with []UserBooking, or 500.
func List(w http.ResponseWriter, r *http.Request) {
	bs, err := dbListUserBookings(r.Context(), db.Pool, middleware.User(r).UserID, "")
	if err != nil {
		log.Printf("list bookings: %v", err)
		response.Error(w, err)
		return
	}
	response.Ok(w, bs, nil)
}

// Get returns one of the signed-in user's bookings by its short reference, for the confirmation page opened from a link.
// Takes the request with the ref in the path; responds 200 with the UserBooking, 404, or 500.
func Get(w http.ResponseWriter, r *http.Request) {
	bs, err := dbListUserBookings(r.Context(), db.Pool, middleware.User(r).UserID, strings.ToUpper(r.PathValue("ref")))
	if err == nil && len(bs) == 0 {
		err = errNoBooking
	}
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Ok(w, bs[0], nil)
}

// Cancel cancels one of the signed-in user's bookings, freeing its seats and refunding it in full, up to an hour
// before the showing. Takes the request with the booking ID in the path; responds 200, or 404 not theirs,
// 409 too close to the showing, 502 the refund failed (the booking is kept), 500.
func Cancel(w http.ResponseWriter, r *http.Request) {
	if err := cancel(r.Context(), r.PathValue("id"), middleware.User(r).UserID); err != nil {
		response.Error(w, err)
		return
	}
	response.Ok(w, nil, nil)
}

// confirm books the seats a succeeded payment paid for, once: calling it again for the same payment returns the same booking.
// A booking that can't be made (seats sold meanwhile, showing started, price changed) is refunded. Takes a context, the
// intent ID, and the event and user it must be for ("" to skip that check, for the webhook); returns the Booking, an
// *AppError for an outcome that's settled, or another error, which is worth retrying.
func confirm(ctx context.Context, piID, eventID, userID string) (Booking, error) {
	pi, err := stripe.GetPaymentIntent(ctx, piID)
	if se, ok := errors.AsType[*stripe.Error](err); ok && se.Status == http.StatusNotFound {
		return Booking{}, errNoPayment
	}
	if err != nil {
		return Booking{}, err
	}
	m := pi.Metadata
	if eventID != "" && (m["event_id"] != eventID || m["user_id"] != userID) {
		return Booking{}, errNoPayment
	}
	if pi.Status != "succeeded" {
		return Booking{}, errNotPaid
	}
	if b, err := dbBookingByPayment(ctx, db.Pool, piID); !errors.Is(err, pgx.ErrNoRows) {
		return b, err
	}
	// Refunded but not booked: the booking failed or was cancelled, so it mustn't be made now.
	if pi.Refunded() {
		return Booking{}, errRefunded
	}

	seatIDs := strings.Split(m["seat_ids"], ",")
	b, err := book(ctx, m["event_id"], m["user_id"], seatIDs, pi)
	if appErr, ok := errors.AsType[*response.AppError](err); ok {
		if err := stripe.Refund(ctx, piID); err != nil {
			return Booking{}, err
		}
		return Booking{}, response.NewError(appErr.Status, appErr.Message+" Your payment has been refunded.")
	}
	if err == nil {
		release(ctx, m["event_id"], m["user_id"], seatIDs)
	}
	return b, err
}

// book makes the booking for a payment and queues the confirmation email in the same transaction. Takes a context, the
// showing, user, seats and payment; returns the Booking (the existing one if this payment was booked meanwhile),
// errNoShowing, errSeatsTaken, errPriceChanged, or the database error.
func book(ctx context.Context, eventID, userID string, seatIDs []string, pi stripe.PaymentIntent) (Booking, error) {
	var b Booking
	err := pgx.BeginFunc(ctx, db.Pool, func(tx pgx.Tx) error {
		f, err := checkSeats(ctx, tx, eventID, seatIDs)
		if err != nil {
			return err
		}
		if pi.Amount != f.TotalCents {
			return errPriceChanged
		}
		if b.BookingID, err = dbCreateBooking(ctx, tx, eventID, f.RoomID, userID, seatIDs, pi.ID); err != nil {
			return err
		}
		b.Ref = refOf(b.BookingID)
		return notification.SendBookingConfirmedNotification(ctx, tx, userID, b.BookingID, b.Ref)
	})
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "23505" {
		if pgErr.ConstraintName == "bookings_payment_intent_id_key" { // the page and the webhook booked it at once
			return dbBookingByPayment(ctx, db.Pool, pi.ID)
		}
		return Booking{}, errSeatsTaken // a seat sold since checkSeats
	}
	return b, err
}

// cancel refunds and deletes the user's booking, which frees its seats. The row stays locked until the refund is done,
// so a booking is never deleted without its refund. Takes a context, booking ID and user ID; returns errNoBooking,
// errTooLate, errRefundFailed, or the database error.
func cancel(ctx context.Context, bookingID, userID string) error {
	return pgx.BeginFunc(ctx, db.Pool, func(tx pgx.Tx) error {
		lb, err := dbLockBooking(ctx, tx, bookingID, userID, time.Now().Add(cancelWindow))
		if pgErr, ok := errors.AsType[*pgconn.PgError](err); errors.Is(err, pgx.ErrNoRows) || ok && pgErr.Code == "22P02" { // 22P02: not a UUID
			return errNoBooking
		}
		if err != nil {
			return err
		}
		if !lb.Cancellable {
			return errTooLate
		}
		if lb.PaymentIntentID != nil {
			if err := stripe.Refund(ctx, *lb.PaymentIntentID); err != nil {
				log.Printf("refund %s: %v", *lb.PaymentIntentID, err)
				return errRefundFailed
			}
		}
		_, err = tx.Exec(ctx, `DELETE FROM bookings WHERE booking_id = $1`, bookingID)
		return err
	})
}

// refOf makes a booking's short reference. The UUIDv7's last 8 hex digits are random, so they make a code that's easy
// to read out; bookings_event_ref_idx keeps it unique within a showing.
func refOf(bookingID string) string {
	return strings.ToUpper(bookingID[len(bookingID)-8:])
}

// checkSeats makes sure the showing hasn't started and every seat is in its room and unbooked. Takes a context, querier,
// event ID and seats; returns the showing's room and the seats' total, errNoShowing, errSeatsTaken, or the database error.
func checkSeats(ctx context.Context, q db.Querier, eventID string, seatIDs []string) (freeSeats, error) {
	f, err := dbFreeSeats(ctx, q, eventID, seatIDs)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); errors.Is(err, pgx.ErrNoRows) || ok && pgErr.Code == "22P02" { // 22P02: not a UUID
		return f, errNoShowing
	}
	if err != nil {
		return f, err
	}
	if f.Free < len(seatIDs) {
		return f, errSeatsTaken
	}
	return f, nil
}
