package resource

import (
	"booking-system/pkg/db"
	"context"

	"github.com/jackc/pgx/v5"
)

// dbListMovies fetches every movie that has a showing in the next 3 days, with those showings. Three days covers the
// frontend's today, tomorrow and the day after in any time zone. Takes a context; returns the movies (featured first,
// then by title) or the database error.
func dbListMovies(ctx context.Context) ([]Movie, error) {
	rows, _ := db.Pool.Query(ctx, `
	SELECT m.movie_id, m.title, m.genre, m.runtime_minutes, m.rating, m.synopsis, m.featured,
	COALESCE(m.poster_path, '') AS poster_path, COALESCE(m.backdrop_path, '') AS backdrop_path,
	COALESCE(m.trailer_key, '') AS trailer_key,
	json_agg(json_build_object(
		'eventID', e.event_id, 'startsAt', e.starts_at, 'format', f.name, 'room', r.name
	) ORDER BY e.starts_at) AS showings
	FROM movies m
	JOIN events e USING (movie_id)
	JOIN event_formats f USING (format_id)
	JOIN rooms r USING (room_id)
	WHERE e.starts_at > now() AND e.starts_at < now() + interval '3 days'
	GROUP BY m.movie_id
	ORDER BY m.featured DESC, m.title
	`)
	return pgx.CollectRows(rows, pgx.RowToStructByName[Movie])
}

// dbListSeats fetches every seat in a showing's room with its price, marking the ones already booked for that showing.
// Takes a context and event ID; returns the seats by row then number, or none if the showing doesn't exist.
func dbListSeats(ctx context.Context, eventID string) ([]Seat, error) {
	rows, _ := db.Pool.Query(ctx, `
	SELECT s.seat_id, s.row_label, s.seat_number, s.seat_type, f.base_price_cents + sts.surcharge_cents AS price_cents,
	bs.seat_id IS NOT NULL AS taken
	FROM events e
	JOIN event_formats f USING (format_id)
	JOIN seats s USING (room_id)
	JOIN seat_type_surcharges sts USING (seat_type)
	LEFT JOIN booking_seats bs ON bs.event_id = e.event_id AND bs.seat_id = s.seat_id
	WHERE e.event_id = @event_id
	ORDER BY s.row_label, s.seat_number
	`, pgx.NamedArgs{"event_id": eventID})
	return pgx.CollectRows(rows, pgx.RowToStructByName[Seat])
}

// dbUnfetchedMovies finds movies with a TMDB ID whose details haven't been fetched yet. Takes a context;
// returns their TMDB IDs keyed by movie ID, or the database error.
func dbUnfetchedMovies(ctx context.Context) (map[string]int, error) {
	rows, _ := db.Pool.Query(ctx, `SELECT movie_id, tmdb_id FROM movies WHERE tmdb_id IS NOT NULL AND trailer_key IS NULL`)
	ids := map[string]int{}
	var movieID string
	var tmdbID int
	_, err := pgx.ForEachRow(rows, []any{&movieID, &tmdbID}, func() error {
		ids[movieID] = tmdbID
		return nil
	})
	return ids, err
}

// dbSetTMDB saves a movie's TMDB details, keeping the seeded synopsis or runtime where TMDB has none.
// Takes a context, the movie ID and the details; returns the database error.
func dbSetTMDB(ctx context.Context, movieID string, m tmdbMovie) error {
	_, err := db.Pool.Exec(ctx, `
	UPDATE movies SET
		synopsis = COALESCE(NULLIF(@overview::text, ''), synopsis),
		runtime_minutes = COALESCE(NULLIF(@runtime::int, 0), runtime_minutes),
		poster_path = @poster,
		backdrop_path = @backdrop,
		trailer_key = @trailer
	WHERE movie_id = @movie_id
	`, pgx.NamedArgs{
		"overview": m.Overview, "runtime": m.Runtime, "poster": m.PosterPath,
		"backdrop": m.BackdropPath, "trailer": m.TrailerKey, "movie_id": movieID,
	})
	return err
}
