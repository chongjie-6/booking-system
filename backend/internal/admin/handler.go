package admin

import (
	"booking-system/pkg/db"
	"booking-system/pkg/response"
	"context"
	"errors"
	"log"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ListMovies returns every film with its upcoming showings. Takes the request; responds 200 with []Movie, or 500.
func ListMovies(w http.ResponseWriter, r *http.Request) {
	movies, err := dbListMovies(r.Context(), db.Pool)
	if err != nil {
		log.Printf("admin list movies: %v", err)
		response.Error(w, err)
		return
	}
	response.Ok(w, movies, nil)
}

// ListRooms returns every screen. Takes the request; responds 200 with []Room, or 500.
func ListRooms(w http.ResponseWriter, r *http.Request) {
	rooms, err := dbListRooms(r.Context(), db.Pool)
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Ok(w, rooms, nil)
}

// CreateMovie adds a film. Takes a movieRequest body; responds 201 with its ID, or 400, 409 TMDB ID in use, 500.
func CreateMovie(w http.ResponseWriter, r *http.Request) {
	saveMovie(w, r, "")
}

// UpdateMovie replaces a film's details. Takes the film ID in the path and a movieRequest body;
// responds 200, or 400, 404, 409 TMDB ID in use, 500.
func UpdateMovie(w http.ResponseWriter, r *http.Request) {
	saveMovie(w, r, r.PathValue("id"))
}

// DeleteMovie deletes a film with no showings. Takes the film ID in the path; responds 200, or 404, 409 has showings, 500.
func DeleteMovie(w http.ResponseWriter, r *http.Request) {
	found, err := dbDeleteMovie(r.Context(), db.Pool, r.PathValue("id"))
	if code(err) == "23503" { // events still reference it
		err = errHasShowings
	} else if code(err) == "22P02" || err == nil && !found {
		err = errNoMovie
	}
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Ok(w, nil, nil)
}

// CreateEvent adds a showing. Takes an eventRequest body; responds 201 with its ID, or 400 bad body, start in the past or
// unknown film or screen, 409 the screen is busy then, 500.
func CreateEvent(w http.ResponseWriter, r *http.Request) {
	saveEvent(w, r, "")
}

// UpdateEvent replaces a showing that has no bookings. Takes the event ID in the path and an eventRequest body;
// responds 200, or 400 as for CreateEvent, 404, 409 has bookings or the screen is busy then, 500.
func UpdateEvent(w http.ResponseWriter, r *http.Request) {
	saveEvent(w, r, r.PathValue("id"))
}

// DeleteEvent deletes a showing that has no bookings. Deleting one with bookings would cascade to them, so it's refused.
// Takes the event ID in the path; responds 200, or 404, 409 has bookings, 500.
func DeleteEvent(w http.ResponseWriter, r *http.Request) {
	ctx, eventID := r.Context(), r.PathValue("id")
	err := pgx.BeginFunc(ctx, db.Pool, func(tx pgx.Tx) error {
		if err := lockUnbooked(ctx, tx, eventID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx, `DELETE FROM events WHERE event_id = $1`, eventID)
		return err
	})
	if err != nil {
		response.Error(w, err)
		return
	}
	response.Ok(w, nil, nil)
}

// saveMovie inserts (movieID "") or replaces a film from the request body and responds as CreateMovie or UpdateMovie.
func saveMovie(w http.ResponseWriter, r *http.Request, movieID string) {
	var req movieRequest
	if err := response.Bind(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	id, err := dbSaveMovie(r.Context(), db.Pool, movieID, req)
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok && pgErr.ConstraintName == "movies_tmdb_id_key" {
		err = errTMDBIDInUse
	} else if errors.Is(err, pgx.ErrNoRows) || code(err) == "22P02" {
		err = errNoMovie
	}
	respond(w, movieID, id, err)
}

// saveEvent inserts (eventID "") or replaces a showing from the request body and responds as CreateEvent or UpdateEvent.
func saveEvent(w http.ResponseWriter, r *http.Request, eventID string) {
	var req eventRequest
	if err := response.Bind(r, &req); err != nil {
		response.Error(w, err)
		return
	}
	if !req.StartsAt.After(time.Now()) {
		response.Error(w, errPast)
		return
	}
	ctx := r.Context()
	var id string
	err := pgx.BeginFunc(ctx, db.Pool, func(tx pgx.Tx) error {
		if eventID != "" {
			if err := lockUnbooked(ctx, tx, eventID); err != nil {
				return err
			}
		}
		overlaps, err := dbOverlaps(ctx, tx, eventID, req.MovieID, req.RoomID, req.StartsAt)
		if err != nil {
			return err
		}
		if overlaps {
			return errOverlap
		}
		id, err = dbSaveEvent(ctx, tx, eventID, req)
		return err
	})
	if code(err) == "23503" { // no such film or room
		err = errBadRef
	}
	respond(w, eventID, id, err)
}

// lockUnbooked locks a showing for changing, and returns errNoEvent, errHasBookings, or the database error.
func lockUnbooked(ctx context.Context, tx pgx.Tx, eventID string) error {
	booked, err := dbLockEvent(ctx, tx, eventID)
	if errors.Is(err, pgx.ErrNoRows) || code(err) == "22P02" {
		return errNoEvent
	}
	if err != nil {
		return err
	}
	if booked {
		return errHasBookings
	}
	return nil
}

// respond sends err, or 201 with the new ID when nothing was being replaced (oldID ""), else 200.
func respond(w http.ResponseWriter, oldID, newID string, err error) {
	switch {
	case err != nil:
		response.Error(w, err)
	case oldID == "":
		response.Created(w, created{newID})
	default:
		response.Ok(w, nil, nil)
	}
}

// code returns err's Postgres error code, or "" if it isn't a Postgres error.
func code(err error) string {
	if pgErr, ok := errors.AsType[*pgconn.PgError](err); ok {
		return pgErr.Code
	}
	return ""
}
