package booking

import (
	"booking-system/pkg/db"
	"booking-system/pkg/rdb"
	"context"
	"log"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/redis/go-redis/v9"
)

// holdTTL is how long chosen seats stay held while the user checks out.
const holdTTL = 10 * time.Minute

// holdScript holds every seat in KEYS for user ARGV[1] for ARGV[2] ms, or none of them if another user holds one.
// The user's own holds are refreshed. Returns 1 on a clash, else 0.
var holdScript = redis.NewScript(`
for _, k in ipairs(KEYS) do
	local h = redis.call('GET', k)
	if h and h ~= ARGV[1] then return 1 end
end
for _, k in ipairs(KEYS) do redis.call('SET', k, ARGV[1], 'PX', ARGV[2]) end
return 0`)

// releaseScript deletes the holds in KEYS that user ARGV[1] owns, leaving other users' holds alone.
var releaseScript = redis.NewScript(`
for _, k in ipairs(KEYS) do
	if redis.call('GET', k) == ARGV[1] then redis.call('DEL', k) end
end
return 0`)

// holdKeys returns the Redis key of each seat's hold for the showing, in seatIDs order.
func holdKeys(eventID string, seatIDs []string) []string {
	keys := make([]string, len(seatIDs))
	for i, s := range seatIDs {
		keys[i] = "hold:" + eventID + ":" + s
	}
	return keys
}

// hold holds the seats for userID, or returns errSeatsTaken if another user holds one. If Redis is down it logs and
// returns nil: booking carries on without holds and booking_seats' unique key stops double sales (see the README).
func hold(ctx context.Context, eventID, userID string, seatIDs []string) error {
	clash, err := holdScript.Run(ctx, rdb.Client, holdKeys(eventID, seatIDs), userID, holdTTL.Milliseconds()).Int()
	if err != nil {
		log.Printf("seat hold skipped: %v", err)
		return nil
	}
	if clash == 1 {
		return errSeatsTaken
	}
	return nil
}

// release drops userID's holds on the seats. A Redis error is ignored, since the holds expire on their own.
func release(ctx context.Context, eventID, userID string, seatIDs []string) {
	releaseScript.Run(ctx, rdb.Client, holdKeys(eventID, seatIDs), userID)
}

// Holders returns which user holds each seat for the showing ("" for none), in seatIDs order, or nil if Redis is down.
func Holders(ctx context.Context, eventID string, seatIDs []string) []string {
	if len(seatIDs) == 0 {
		return nil
	}
	vals, err := rdb.Client.MGet(ctx, holdKeys(eventID, seatIDs)...).Result()
	if err != nil {
		return nil
	}
	holders := make([]string, len(vals))
	for i, v := range vals {
		holders[i], _ = v.(string)
	}
	return holders
}

// dbFreeSeats counts the seats in seatIDs that are in the showing's room and not booked yet, with the room and seat price.
// Returns pgx.ErrNoRows if there's no showing with that ID still to start.
func dbFreeSeats(ctx context.Context, q db.Querier, eventID string, seatIDs []string) (freeSeats, error) {
	rows, _ := q.Query(ctx, `
	SELECT e.room_id, e.price_cents, count(s.seat_id) FILTER (WHERE bs.seat_id IS NULL) AS free
	FROM events e
	LEFT JOIN seats s ON s.room_id = e.room_id AND s.seat_id = ANY(@seat_ids::uuid[])
	LEFT JOIN booking_seats bs ON bs.event_id = e.event_id AND bs.seat_id = s.seat_id
	WHERE e.event_id = @event_id AND e.starts_at > now()
	GROUP BY e.room_id, e.price_cents
	`, pgx.NamedArgs{"event_id": eventID, "seat_ids": seatIDs})
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[freeSeats])
}

// dbCreateBooking inserts a booking and its seats. Takes the transaction, showing, its room, user, seats and the Stripe
// payment ("" for none); returns the new booking ID, or the database error: a 23505 on booking_seats if a seat was sold
// meanwhile, or on bookings_payment_intent_id_key if the payment is already booked.
func dbCreateBooking(ctx context.Context, tx pgx.Tx, eventID, roomID, userID string, seatIDs []string, paymentIntentID string) (string, error) {
	var bookingID string
	err := tx.QueryRow(ctx, `
	INSERT INTO bookings (event_id, user_id, payment_intent_id) VALUES (@event_id, @user_id, NULLIF(@payment_intent_id, ''))
	RETURNING booking_id
	`, pgx.NamedArgs{"event_id": eventID, "user_id": userID, "payment_intent_id": paymentIntentID}).Scan(&bookingID)
	if err != nil {
		return "", err
	}
	_, err = tx.Exec(ctx, `
	INSERT INTO booking_seats (booking_id, event_id, room_id, seat_id)
	SELECT @booking_id, @event_id, @room_id, unnest(@seat_ids::uuid[])
	`, pgx.NamedArgs{"booking_id": bookingID, "event_id": eventID, "room_id": roomID, "seat_ids": seatIDs})
	return bookingID, err
}

// dbBookingByPayment finds the booking a Stripe payment made. Takes a context, querier and intent ID;
// returns the Booking, or pgx.ErrNoRows if the payment hasn't been booked.
func dbBookingByPayment(ctx context.Context, q db.Querier, paymentIntentID string) (Booking, error) {
	rows, _ := q.Query(ctx, `
	SELECT booking_id, upper(right(booking_id::text, 8)) AS ref FROM bookings WHERE payment_intent_id = @payment_intent_id
	`, pgx.NamedArgs{"payment_intent_id": paymentIntentID})
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByPos[Booking])
}

// dbLockBooking locks one of the user's bookings for cancelling, until the transaction ends. Takes the transaction,
// booking ID, user ID, and the latest start time that can't be cancelled any more; returns the booking's payment and
// whether its showing starts after cutoff, or pgx.ErrNoRows if the user has no such booking.
func dbLockBooking(ctx context.Context, tx pgx.Tx, bookingID, userID string, cutoff time.Time) (lockedBooking, error) {
	rows, _ := tx.Query(ctx, `
	SELECT b.payment_intent_id, e.starts_at > @cutoff AS cancellable
	FROM bookings b JOIN events e ON e.event_id = b.event_id
	WHERE b.booking_id = @booking_id AND b.user_id = @user_id
	FOR UPDATE OF b
	`, pgx.NamedArgs{"booking_id": bookingID, "user_id": userID, "cutoff": cutoff})
	return pgx.CollectExactlyOneRow(rows, pgx.RowToStructByName[lockedBooking])
}

// dbListUserBookings fetches the user's bookings, latest showing first, each with its film, screen, price and seats.
// Ref is worked out as in refOf. Takes a context, querier, user ID and a ref to fetch just that booking ("" for all);
// returns the bookings or the database error.
func dbListUserBookings(ctx context.Context, q db.Querier, userID, ref string) ([]UserBooking, error) {
	rows, _ := q.Query(ctx, `
	SELECT b.booking_id, upper(right(b.booking_id::text, 8)) AS ref, e.event_id, m.title, e.starts_at,
	r.name AS room, e.price_cents,
	json_agg(json_build_object('row', s.row_label, 'number', s.seat_number) ORDER BY s.row_label, s.seat_number) AS seats
	FROM bookings b
	JOIN events e ON e.event_id = b.event_id
	JOIN movies m ON m.movie_id = e.movie_id
	JOIN rooms r ON r.room_id = e.room_id
	JOIN booking_seats bs ON bs.booking_id = b.booking_id
	JOIN seats s ON s.seat_id = bs.seat_id
	WHERE b.user_id = @user_id AND (@ref = '' OR upper(right(b.booking_id::text, 8)) = @ref)
	GROUP BY b.booking_id, e.event_id, m.movie_id, r.room_id
	ORDER BY e.starts_at DESC, b.booking_id DESC
	`, pgx.NamedArgs{"user_id": userID, "ref": ref})
	return pgx.CollectRows(rows, pgx.RowToStructByName[UserBooking])
}
