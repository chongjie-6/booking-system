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

	var room, user, format string
	db.Pool.QueryRow(ctx, `INSERT INTO rooms (name, capacity) VALUES ('Admin test screen', 1) RETURNING room_id`).Scan(&room)
	db.Pool.QueryRow(ctx, `INSERT INTO event_formats (name, base_price_cents) VALUES ('Admin test', 1500) RETURNING format_id`).Scan(&format)
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
		db.Pool.Exec(ctx, `DELETE FROM event_formats WHERE format_id = $1`, format)
	})

	at := time.Now().Add(24 * time.Hour).Truncate(time.Minute)
	showing := func(start time.Time) eventRequest {
		return eventRequest{MovieID: movie, RoomID: room, StartsAt: start, FormatID: format}
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
	if status, _ := call(DeleteFormat, http.MethodDelete, format, nil); status != http.StatusConflict {
		t.Fatalf("delete format with showings = %d, want 409", status)
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

// TestFormatsAndSurcharges checks formats can be added, renamed, repriced and deleted, that names are unique and prices
// are at least Stripe's 50 cents, and that a seat type's surcharge can be set. Needs DATABASE_URL. Handlers commit, so
// the format is deleted and the surcharge put back after.
func TestFormatsAndSurcharges(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	db.Connect(ctx)
	db.Migrate()

	call := func(h http.HandlerFunc, method, key, id string, body any) (int, string) {
		t.Helper()
		b, _ := json.Marshal(body)
		r := httptest.NewRequest(method, "/", strings.NewReader(string(b)))
		r.SetPathValue(key, id)
		w := httptest.NewRecorder()
		h(w, r)
		var res struct{ Data created }
		json.Unmarshal(w.Body.Bytes(), &res)
		return w.Code, res.Data.ID
	}

	status, id := call(CreateFormat, http.MethodPost, "id", "", formatRequest{Name: "Admin test format", BasePriceCents: 1500})
	if status != http.StatusCreated {
		t.Fatalf("create format = %d", status)
	}
	t.Cleanup(func() { db.Pool.Exec(ctx, `DELETE FROM event_formats WHERE format_id = $1`, id) })
	if status, _ := call(CreateFormat, http.MethodPost, "id", "", formatRequest{Name: "Admin test format", BasePriceCents: 1500}); status != http.StatusConflict {
		t.Fatalf("duplicate name = %d, want 409", status)
	}
	if status, _ := call(CreateFormat, http.MethodPost, "id", "", formatRequest{Name: "Too cheap", BasePriceCents: 49}); status != http.StatusBadRequest {
		t.Fatalf("price under 50 cents = %d, want 400", status)
	}
	if status, _ := call(UpdateFormat, http.MethodPut, "id", id, formatRequest{Name: "Admin test format 2", BasePriceCents: 1800}); status != http.StatusOK {
		t.Fatalf("update format = %d, want 200", status)
	}
	if status, _ := call(UpdateFormat, http.MethodPut, "id", "not-a-uuid", formatRequest{Name: "X", BasePriceCents: 1800}); status != http.StatusNotFound {
		t.Fatalf("update unknown format = %d, want 404", status)
	}
	formats, err := dbListFormats(ctx, db.Pool)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, f := range formats {
		found = found || f == Format{id, "Admin test format 2", 1800}
	}
	if !found {
		t.Fatalf("formats = %+v, want the updated one", formats)
	}
	if status, _ := call(DeleteFormat, http.MethodDelete, "id", id, nil); status != http.StatusOK {
		t.Fatalf("delete unused format = %d, want 200", status)
	}
	if status, _ := call(DeleteFormat, http.MethodDelete, "id", id, nil); status != http.StatusNotFound {
		t.Fatalf("delete it again = %d, want 404", status)
	}

	var was int
	if err := db.Pool.QueryRow(ctx, `SELECT surcharge_cents FROM seat_type_surcharges WHERE seat_type = 'companion'`).Scan(&was); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		db.Pool.Exec(ctx, `UPDATE seat_type_surcharges SET surcharge_cents = $1 WHERE seat_type = 'companion'`, was)
	})
	if status, _ := call(UpdateSurcharge, http.MethodPut, "type", "companion", surchargeRequest{SurchargeCents: was + 1}); status != http.StatusOK {
		t.Fatalf("set surcharge = %d, want 200", status)
	}
	surcharges, err := dbListSurcharges(ctx, db.Pool)
	if err != nil || len(surcharges) == 0 || surcharges[0].SeatType != "standard" {
		t.Fatalf("surcharges = %+v, %v; want them in seat_type order", surcharges, err)
	}
	for _, s := range surcharges {
		if s.SeatType == "companion" && s.SurchargeCents != was+1 {
			t.Fatalf("companion surcharge = %d, want %d", s.SurchargeCents, was+1)
		}
	}
	if status, _ := call(UpdateSurcharge, http.MethodPut, "type", "balcony", surchargeRequest{SurchargeCents: 100}); status != http.StatusNotFound {
		t.Fatalf("unknown seat type = %d, want 404", status)
	}
	if status, _ := call(UpdateSurcharge, http.MethodPut, "type", "companion", surchargeRequest{SurchargeCents: -1}); status != http.StatusBadRequest {
		t.Fatalf("negative surcharge = %d, want 400", status)
	}
}
