package admin

import (
	"booking-system/pkg/db"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// TestShowings checks films and showings can be added and changed, that a screen can't be double-booked, that showings
// must be in the future, and that a showing with bookings (or a film with showings) can't be changed or deleted.
// Needs DATABASE_URL. Handlers commit, so the rows are deleted after.
func TestShowings(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	db.Connect(ctx)
	db.Migrate()

	call := func(h http.HandlerFunc, method, id string, body any) (int, string) {
		t.Helper()
		b, _ := json.Marshal(body)
		r := httptest.NewRequest(method, "/", strings.NewReader(string(b)))
		r.SetPathValue("id", id)
		w := httptest.NewRecorder()
		h(w, r)
		var res struct{ Data created }
		json.Unmarshal(w.Body.Bytes(), &res)
		return w.Code, res.Data.ID
	}

	var room, user string
	db.Pool.QueryRow(ctx, `INSERT INTO rooms (name, capacity) VALUES ('Admin test screen', 1) RETURNING room_id`).Scan(&room)
	db.Pool.QueryRow(ctx, `INSERT INTO users (first_name, last_name, email) VALUES ('A', 'A', 'a@admin.test') RETURNING user_id`).Scan(&user)
	status, movie := call(CreateMovie, http.MethodPost, "", movieRequest{Title: "Admin test", Genre: "Drama", RuntimeMinutes: 90, Rating: "M"})
	if status != http.StatusCreated {
		t.Fatalf("create movie = %d", status)
	}
	t.Cleanup(func() {
		db.Pool.Exec(ctx, `DELETE FROM users WHERE user_id = $1`, user)
		db.Pool.Exec(ctx, `DELETE FROM events WHERE room_id = $1`, room)
		db.Pool.Exec(ctx, `DELETE FROM rooms WHERE room_id = $1`, room)
		db.Pool.Exec(ctx, `DELETE FROM movies WHERE movie_id = $1`, movie)
	})

	at := time.Now().Add(24 * time.Hour).Truncate(time.Minute)
	showing := func(start time.Time) eventRequest {
		return eventRequest{MovieID: movie, RoomID: room, StartsAt: start, PriceCents: 1500}
	}
	status, first := call(CreateEvent, http.MethodPost, "", showing(at))
	if status != http.StatusCreated {
		t.Fatalf("create showing = %d", status)
	}
	if status, _ := call(CreateEvent, http.MethodPost, "", showing(at.Add(89*time.Minute))); status != http.StatusConflict {
		t.Fatalf("showing during another = %d, want 409", status)
	}
	if status, _ := call(CreateEvent, http.MethodPost, "", showing(at.Add(-89*time.Minute))); status != http.StatusConflict {
		t.Fatalf("showing running into another = %d, want 409", status)
	}
	status, second := call(CreateEvent, http.MethodPost, "", showing(at.Add(90*time.Minute)))
	if status != http.StatusCreated {
		t.Fatalf("showing straight after another = %d, want 201", status)
	}
	if status, _ := call(CreateEvent, http.MethodPost, "", showing(time.Now().Add(-time.Hour))); status != http.StatusBadRequest {
		t.Fatalf("showing in the past = %d, want 400", status)
	}
	bad := showing(at.Add(48 * time.Hour))
	bad.RoomID = movie // a UUID, but not a room's
	if status, _ := call(CreateEvent, http.MethodPost, "", bad); status != http.StatusBadRequest {
		t.Fatalf("showing in an unknown room = %d, want 400", status)
	}
	if status, _ := call(UpdateEvent, http.MethodPut, first, showing(at.Add(-2*time.Hour))); status != http.StatusOK {
		t.Fatalf("move showing earlier = %d, want 200", status)
	}
	if status, _ := call(UpdateEvent, http.MethodPut, first, showing(at.Add(time.Hour))); status != http.StatusConflict {
		t.Fatalf("move showing onto another = %d, want 409", status)
	}

	if _, err := db.Pool.Exec(ctx, `INSERT INTO bookings (event_id, user_id) VALUES ($1, $2)`, first, user); err != nil {
		t.Fatal(err)
	}
	if status, _ := call(UpdateEvent, http.MethodPut, first, showing(at.Add(-3*time.Hour))); status != http.StatusConflict {
		t.Fatalf("change booked showing = %d, want 409", status)
	}
	if status, _ := call(DeleteEvent, http.MethodDelete, first, nil); status != http.StatusConflict {
		t.Fatalf("delete booked showing = %d, want 409", status)
	}
	if status, _ := call(DeleteMovie, http.MethodDelete, movie, nil); status != http.StatusConflict {
		t.Fatalf("delete film with showings = %d, want 409", status)
	}
	if status, _ := call(DeleteEvent, http.MethodDelete, second, nil); status != http.StatusOK {
		t.Fatalf("delete unbooked showing = %d, want 200", status)
	}
	if status, _ := call(DeleteEvent, http.MethodDelete, second, nil); status != http.StatusNotFound {
		t.Fatalf("delete it again = %d, want 404", status)
	}

	movies, err := dbListMovies(ctx, db.Pool)
	if err != nil {
		t.Fatal(err)
	}
	for _, m := range movies {
		if m.MovieID == movie && (len(m.Showings) != 1 || m.Showings[0].EventID != first || m.Showings[0].Room != "Admin test screen") {
			t.Fatalf("listed showings = %+v, want just the booked one", m.Showings)
		}
	}
}
