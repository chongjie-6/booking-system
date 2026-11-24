package booking

import (
	"booking-system/pkg/db"
	"context"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"
)

// TestListUserBookings checks a user sees only their own bookings, latest showing first, with refs, seats and what they
// paid, which a later price change doesn't alter.
// Runs only when DATABASE_URL is set (CI gives it a Postgres); everything happens in a transaction that's rolled back.
func TestListUserBookings(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	db.Connect(ctx)
	db.Migrate()
	tx, err := db.Pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)

	one := func(sql string, args ...any) string {
		t.Helper()
		var id string
		if err := tx.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		return id
	}
	alice := one(`INSERT INTO users (first_name, last_name, email) VALUES ('A', 'A', 'alice@list.test') RETURNING user_id`)
	bob := one(`INSERT INTO users (first_name, last_name, email) VALUES ('B', 'B', 'bob@list.test') RETURNING user_id`)
	movie := one(`INSERT INTO movies (title, genre, runtime_minutes, rating) VALUES ('Film', 'Drama', 90, 'M') RETURNING movie_id`)
	room := one(`INSERT INTO rooms (name, capacity) VALUES ('List test screen', 3) RETURNING room_id`)
	seats := map[string]string{}
	for _, n := range []int{1, 2} {
		seats["A"+strconv.Itoa(n)] = one(`INSERT INTO seats (room_id, row_label, seat_number) VALUES ($1, 'A', $2) RETURNING seat_id`, room, n)
	}
	seats["A3"] = one(`INSERT INTO seats (room_id, row_label, seat_number, seat_type) VALUES ($1, 'A', 3, 'premium') RETURNING seat_id`, room)
	if _, err := tx.Exec(ctx, `UPDATE seat_type_surcharges SET surcharge_cents = 500 WHERE seat_type = 'premium'`); err != nil {
		t.Fatal(err)
	}
	event := func(starts time.Time, price int) string {
		format := one(`INSERT INTO event_formats (name, base_price_cents) VALUES ($1, $2) RETURNING format_id`,
			"List test "+strconv.Itoa(price), price)
		return one(`INSERT INTO events (starts_at, room_id, movie_id, format_id) VALUES ($1, $2, $3, $4) RETURNING event_id`,
			starts, room, movie, format)
	}
	soon, later := event(time.Now().Add(time.Hour), 1500), event(time.Now().Add(48*time.Hour), 2000)
	book := func(eventID, userID string, seatLabels ...string) {
		ids := make([]string, len(seatLabels))
		for i, l := range seatLabels {
			ids[i] = seats[l]
		}
		if _, err := dbCreateBooking(ctx, tx, eventID, room, userID, ids, ""); err != nil {
			t.Fatal(err)
		}
	}
	book(soon, alice, "A3", "A1")
	book(later, alice, "A2")
	book(soon, bob, "A2")
	// Bookings keep what they sold for.
	if _, err := tx.Exec(ctx, `UPDATE event_formats SET base_price_cents = 9900; UPDATE seat_type_surcharges SET surcharge_cents = 0`); err != nil {
		t.Fatal(err)
	}

	got, err := dbListUserBookings(ctx, tx, alice, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d bookings, want 2 (bob's left out)", len(got))
	}
	if got[0].EventID != later || got[1].EventID != soon {
		t.Errorf("order = %s, %s; want latest showing first", got[0].EventID, got[1].EventID)
	}
	b := got[1]
	if want := strings.ToUpper(b.BookingID[len(b.BookingID)-8:]); b.Ref != want {
		t.Errorf("ref = %q, want %q as Create makes it", b.Ref, want)
	}
	if b.Title != "Film" || b.Room != "List test screen" || b.TotalCents != 1500+1500+500 { // A3 is premium
		t.Errorf("booking = %+v", b)
	}
	if len(b.Seats) != 2 || b.Seats[0] != (bookedSeat{"A", 1}) || b.Seats[1] != (bookedSeat{"A", 3}) {
		t.Errorf("seats = %v, want A1 then A3", b.Seats)
	}

	if got, err := dbListUserBookings(ctx, tx, one(`SELECT uuidv7()::text`), ""); err != nil || got == nil || len(got) != 0 {
		t.Errorf("no bookings = %v, %v; want an empty, non-nil slice so the API sends []", got, err)
	}
}
