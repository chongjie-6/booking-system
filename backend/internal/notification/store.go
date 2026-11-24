package notification

import (
	"booking-system/pkg/db"
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// dbInsertResetToken saves a password reset token's hash. Takes a context, the hash, user ID and expiry; returns the database error, if any.
func dbInsertResetToken(ctx context.Context, tokenHash string, userID string, expiresAt time.Time) error {
	_, err := db.Pool.Exec(ctx, `
	INSERT INTO password_reset_tokens (token_hash, user_id, expires_at)
	VALUES (@token_hash, @user_id, @expires_at)
	`, pgx.NamedArgs{"token_hash": tokenHash, "user_id": userID, "expires_at": expiresAt})
	return err
}

// ticket is what a booking confirmation email shows. When is in the cinema's time zone, e.g. "Tue 18 Nov, 7:00 pm".
type ticket struct {
	Title      string
	When       string
	Room       string
	Format     string // e.g. "IMAX"
	Seats      string // e.g. "C4, C5"
	TotalCents int
}

// dbGetTicket fetches a booking's film, time, screen, format, seats and total. Takes a context and booking ID; returns the ticket or the database error.
// ponytail: the cinema's time zone is fixed to Sydney, like the dev seed; store it on rooms if cinemas in other zones join.
func dbGetTicket(ctx context.Context, bookingID string) (ticket, error) {
	var t ticket
	err := db.Pool.QueryRow(ctx, `
	SELECT m.title,
	to_char(e.starts_at AT TIME ZONE 'Australia/Sydney', 'Dy FMDD Mon, FMHH12:MI am'),
	r.name,
	f.name,
	string_agg(s.row_label || s.seat_number, ', ' ORDER BY s.row_label, s.seat_number),
	sum(bs.price_cents)
	FROM bookings b
	JOIN events e ON e.event_id = b.event_id
	JOIN movies m ON m.movie_id = e.movie_id
	JOIN rooms r ON r.room_id = e.room_id
	JOIN event_formats f ON f.format_id = e.format_id
	JOIN booking_seats bs ON bs.booking_id = b.booking_id
	JOIN seats s ON s.seat_id = bs.seat_id
	WHERE b.booking_id = @booking_id
	GROUP BY m.title, e.starts_at, r.name, f.name
	`, pgx.NamedArgs{"booking_id": bookingID}).Scan(&t.Title, &t.When, &t.Room, &t.Format, &t.Seats, &t.TotalCents)
	return t, err
}
