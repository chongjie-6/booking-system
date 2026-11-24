package admin

import (
	"booking-system/pkg/db"
	"context"
	"time"

	"github.com/jackc/pgx/v5"
)

// dbListMovies fetches every film, by title, with its upcoming showings soonest first and how many seats each has sold.
// Takes a context and querier; returns the films or the database error.
func dbListMovies(ctx context.Context, q db.Querier) ([]Movie, error) {
	rows, _ := q.Query(ctx, `
	SELECT m.movie_id, m.title, m.genre, m.runtime_minutes, m.rating, m.synopsis, m.tmdb_id, m.featured,
	COALESCE(json_agg(json_build_object(
		'eventID', e.event_id, 'startsAt', e.starts_at, 'formatID', e.format_id, 'format', f.name,
		'roomID', e.room_id, 'room', c.name || ' · ' || r.name,
		'booked', (SELECT count(*) FROM booking_seats bs WHERE bs.event_id = e.event_id)
	) ORDER BY e.starts_at) FILTER (WHERE e.event_id IS NOT NULL), '[]') AS showings
	FROM movies m
	LEFT JOIN events e ON e.movie_id = m.movie_id AND e.starts_at > now()
	LEFT JOIN event_formats f ON f.format_id = e.format_id
	LEFT JOIN rooms r ON r.room_id = e.room_id
	LEFT JOIN cinemas c ON c.cinema_id = r.cinema_id
	GROUP BY m.movie_id
	ORDER BY m.title
	`)
	return pgx.CollectRows(rows, pgx.RowToStructByName[Movie])
}

// dbListRooms fetches every screen, named with its cinema, by cinema then name. Takes a context and querier; returns the rooms or the database error.
func dbListRooms(ctx context.Context, q db.Querier) ([]Room, error) {
	rows, _ := q.Query(ctx, `
	SELECT r.room_id, c.name || ' · ' || r.name AS name, r.capacity FROM rooms r JOIN cinemas c USING (cinema_id) ORDER BY c.name, r.name
	`)
	return pgx.CollectRows(rows, pgx.RowToStructByName[Room])
}

// dbListFormats fetches every showing format, cheapest first. Takes a context and querier; returns the formats or the
// database error.
func dbListFormats(ctx context.Context, q db.Querier) ([]Format, error) {
	rows, _ := q.Query(ctx, `SELECT format_id, name, base_price_cents FROM event_formats ORDER BY base_price_cents, name`)
	return pgx.CollectRows(rows, pgx.RowToStructByName[Format])
}

// dbSaveFormat inserts a format (formatID "") or replaces one. Takes a context, querier, format ID and request; returns
// the format's ID, pgx.ErrNoRows if there's no such format, or the database error (23505 if the name is taken).
func dbSaveFormat(ctx context.Context, q db.Querier, formatID string, f formatRequest) (string, error) {
	sql := `INSERT INTO event_formats (name, base_price_cents) VALUES (@name, @price) RETURNING format_id`
	if formatID != "" {
		sql = `UPDATE event_formats SET name = @name, base_price_cents = @price WHERE format_id = @format_id RETURNING format_id`
	}
	rows, _ := q.Query(ctx, sql, pgx.NamedArgs{"format_id": formatID, "name": f.Name, "price": f.BasePriceCents})
	return pgx.CollectExactlyOneRow(rows, pgx.RowTo[string])
}

// dbDeleteFormat deletes a format. Takes a context, querier and format ID; returns whether it existed, or the database
// error (23503 if any showing, past or upcoming, uses it).
func dbDeleteFormat(ctx context.Context, q db.Querier, formatID string) (bool, error) {
	tag, err := q.Exec(ctx, `DELETE FROM event_formats WHERE format_id = $1`, formatID)
	return tag.RowsAffected() == 1, err
}

// dbListSurcharges fetches what each seat type adds to a seat's price, in seat_type order. Takes a context and querier;
// returns the surcharges or the database error.
func dbListSurcharges(ctx context.Context, q db.Querier) ([]Surcharge, error) {
	rows, _ := q.Query(ctx, `SELECT seat_type, surcharge_cents FROM seat_type_surcharges ORDER BY seat_type`)
	return pgx.CollectRows(rows, pgx.RowToStructByName[Surcharge])
}

// dbSetSurcharge sets what a seat type adds to a seat's price. Takes a context, querier, seat type and cents; returns
// whether the seat type exists, or the database error.
func dbSetSurcharge(ctx context.Context, q db.Querier, seatType string, cents int) (bool, error) {
	// Compared as text, so an unknown type is "not found" rather than an invalid enum value.
	tag, err := q.Exec(ctx, `UPDATE seat_type_surcharges SET surcharge_cents = $2 WHERE seat_type::text = $1`, seatType, cents)
	return tag.RowsAffected() == 1, err
}

// dbSaveMovie inserts a film (movieID "") or replaces one. A changed TMDB ID clears the fetched details, so
// resource.FillFromTMDB fetches them again. Takes a context, querier, film ID and request; returns the film's ID,
// pgx.ErrNoRows if there's no such film, or the database error (23505 on movies_tmdb_id_key if the TMDB ID is taken).
func dbSaveMovie(ctx context.Context, q db.Querier, movieID string, m movieRequest) (string, error) {
	sql := `
	INSERT INTO movies (title, genre, runtime_minutes, rating, synopsis, tmdb_id, featured)
	VALUES (@title, @genre, @runtime, @rating, @synopsis, @tmdb_id, @featured)
	RETURNING movie_id`
	if movieID != "" {
		sql = `
		UPDATE movies SET title = @title, genre = @genre, runtime_minutes = @runtime, rating = @rating,
		synopsis = @synopsis, featured = @featured, tmdb_id = @tmdb_id,
		poster_path = CASE WHEN tmdb_id IS DISTINCT FROM @tmdb_id THEN NULL ELSE poster_path END,
		backdrop_path = CASE WHEN tmdb_id IS DISTINCT FROM @tmdb_id THEN NULL ELSE backdrop_path END,
		trailer_key = CASE WHEN tmdb_id IS DISTINCT FROM @tmdb_id THEN NULL ELSE trailer_key END
		WHERE movie_id = @movie_id
		RETURNING movie_id`
	}
	rows, _ := q.Query(ctx, sql, pgx.NamedArgs{
		"movie_id": movieID, "title": m.Title, "genre": m.Genre, "runtime": m.RuntimeMinutes, "rating": m.Rating,
		"synopsis": m.Synopsis, "tmdb_id": m.TMDBID, "featured": m.Featured,
	})
	return pgx.CollectExactlyOneRow(rows, pgx.RowTo[string])
}

// dbDeleteMovie deletes a film. Takes a context, querier and film ID; returns whether it existed, or the database
// error (23503 if it still has showings).
func dbDeleteMovie(ctx context.Context, q db.Querier, movieID string) (bool, error) {
	tag, err := q.Exec(ctx, `DELETE FROM movies WHERE movie_id = $1`, movieID)
	return tag.RowsAffected() == 1, err
}

// dbLockEvent locks a showing until the transaction ends, which also holds off new bookings for it (their foreign keys
// wait on the lock). Takes the transaction and event ID; returns whether it has bookings, pgx.ErrNoRows if there's no
// such showing, or the database error.
func dbLockEvent(ctx context.Context, tx pgx.Tx, eventID string) (bool, error) {
	if _, err := tx.Exec(ctx, `SELECT 1 FROM events WHERE event_id = $1 FOR UPDATE`, eventID); err != nil {
		return false, err
	}
	// A separate statement, so it sees bookings that committed while it waited for the lock.
	var exists, booked bool
	err := tx.QueryRow(ctx, `
	SELECT EXISTS (SELECT 1 FROM events WHERE event_id = $1), EXISTS (SELECT 1 FROM bookings WHERE event_id = $1)
	`, eventID).Scan(&exists, &booked)
	if err == nil && !exists {
		err = pgx.ErrNoRows
	}
	return booked, err
}

// dbOverlaps reports whether the screen has another showing (not eventID, "" for a new one) that would still be running
// when this film starts, or start before it ends. Takes a context, querier, event ID, film, room and start time.
func dbOverlaps(ctx context.Context, q db.Querier, eventID, movieID, roomID string, startsAt time.Time) (bool, error) {
	rows, _ := q.Query(ctx, `
	SELECT EXISTS (
		SELECT 1 FROM events e JOIN movies m ON m.movie_id = e.movie_id
		WHERE e.room_id = @room_id AND e.event_id IS DISTINCT FROM NULLIF(@event_id, '')::uuid
		AND e.starts_at < @starts_at::timestamptz
			+ make_interval(mins => (SELECT runtime_minutes FROM movies WHERE movie_id = @movie_id))
		AND @starts_at::timestamptz < e.starts_at + make_interval(mins => m.runtime_minutes)
	)
	`, pgx.NamedArgs{"event_id": eventID, "movie_id": movieID, "room_id": roomID, "starts_at": startsAt})
	return pgx.CollectExactlyOneRow(rows, pgx.RowTo[bool])
}

// dbSaveEvent inserts a showing (eventID "") or replaces one. Takes a context, querier, event ID and request; returns the
// showing's ID, or the database error (23503 if the film, room or format doesn't exist).
func dbSaveEvent(ctx context.Context, q db.Querier, eventID string, e eventRequest) (string, error) {
	sql := `
	INSERT INTO events (movie_id, room_id, starts_at, format_id) VALUES (@movie_id, @room_id, @starts_at, @format_id)
	RETURNING event_id`
	if eventID != "" {
		sql = `
		UPDATE events SET movie_id = @movie_id, room_id = @room_id, starts_at = @starts_at, format_id = @format_id
		WHERE event_id = @event_id
		RETURNING event_id`
	}
	rows, _ := q.Query(ctx, sql, pgx.NamedArgs{
		"event_id": eventID, "movie_id": e.MovieID, "room_id": e.RoomID, "starts_at": e.StartsAt, "format_id": e.FormatID,
	})
	return pgx.CollectExactlyOneRow(rows, pgx.RowTo[string])
}
