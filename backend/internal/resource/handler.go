package resource

import (
	"booking-system/internal/booking"
	"booking-system/pkg/response"
	"errors"
	"net/http"

	"github.com/jackc/pgx/v5/pgconn"
)

// ListMovies returns the movies showing in the next 3 days with their showings. Takes the request; responds 200 with the movies, or 500.
func ListMovies(w http.ResponseWriter, r *http.Request) {
	movies, err := dbListMovies(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Ok(w, movies, nil)
}

// ComingSoon returns the movies whose first showing is after ListMovies's 3 days, with their showings. Takes the request;
// responds 200 with the movies, or 500.
func ComingSoon(w http.ResponseWriter, r *http.Request) {
	movies, err := dbComingSoon(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Ok(w, movies, nil)
}

// ListCinemas returns every cinema. Takes the request; responds 200 with the cinemas by name, or 500.
func ListCinemas(w http.ResponseWriter, r *http.Request) {
	cinemas, err := dbListCinemas(r.Context())
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Ok(w, cinemas, nil)
}

// ListSeats returns a showing's seats and which are taken. Takes the request with the event ID in the path;
// responds 200 with the seats, 404 if there's no such showing (or the ID isn't a UUID), or 500.
func ListSeats(w http.ResponseWriter, r *http.Request) {
	seats, err := dbListSeats(r.Context(), r.PathValue("id"))
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.Code == "22P02" { // invalid_text_representation: not a UUID
		response.Error(w, response.ErrNotFound)
		return
	}
	if err != nil {
		response.Error(w, err)
		return
	}
	if len(seats) == 0 {
		response.Error(w, response.ErrNotFound)
		return
	}
	// Seats someone is checking out with show as taken. that includes the viewer's own holds; the frontend
	// releases them before going back to the seat map, so look up the viewer here only if that stops being enough.
	ids := make([]string, len(seats))
	for i, s := range seats {
		ids[i] = s.SeatID
	}
	for i, h := range booking.Holders(r.Context(), r.PathValue("id"), ids) {
		seats[i].Taken = seats[i].Taken || h != ""
	}
	response.Ok(w, seats, nil)
}
