package resource

import (
	"booking-system/pkg/db"
	"context"

	"github.com/jackc/pgx/v5"
)

// dbListMovies fetches every movie that has a showing in the next 3 days, with those showings. Three days covers the
// frontend's today, tomorrow and the day after in any time zone. Takes a context; returns the movies (by title) or the database error.
func dbListMovies(ctx context.Context) ([]Movie, error) {
	rows, _ := db.Pool.Query(ctx, `
	SELECT m.movie_id, m.title, m.genre, m.runtime_minutes, m.rating, m.synopsis,
	json_agg(json_build_object(
		'eventID', e.event_id, 'startsAt', e.starts_at, 'priceCents', e.price_cents, 'room', r.name
	) ORDER BY e.starts_at) AS showings
	FROM movies m
	JOIN events e USING (movie_id)
	JOIN rooms r USING (room_id)
	WHERE e.starts_at > now() AND e.starts_at < now() + interval '3 days'
	GROUP BY m.movie_id
	ORDER BY m.title
	`)
	return pgx.CollectRows(rows, pgx.RowToStructByName[Movie])
}

// dbListSeats fetches every seat in a showing's room, marking the ones already booked for that showing.
// Takes a context and event ID; returns the seats by row then number, or none if the showing doesn't exist.
func dbListSeats(ctx context.Context, eventID string) ([]Seat, error) {
	rows, _ := db.Pool.Query(ctx, `
	SELECT s.seat_id, s.row_label, s.seat_number, s.seat_type, bs.seat_id IS NOT NULL AS taken
	FROM events e
	JOIN seats s USING (room_id)
	LEFT JOIN booking_seats bs ON bs.event_id = e.event_id AND bs.seat_id = s.seat_id
	WHERE e.event_id = @event_id
	ORDER BY s.row_label, s.seat_number
	`, pgx.NamedArgs{"event_id": eventID})
	return pgx.CollectRows(rows, pgx.RowToStructByName[Seat])
}
