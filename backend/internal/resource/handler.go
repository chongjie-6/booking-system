package resource

import (
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
	response.Ok(w, seats, nil)
}
