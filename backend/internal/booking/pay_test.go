package booking

import (
	"booking-system/pkg/db"
	"booking-system/pkg/rdb"
	"booking-system/pkg/response"
	"booking-system/pkg/stripe"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeStripe serves GET /payment_intents/{id} from intents and records POST /refunds, marking the intent refunded.
type fakeStripe struct {
	mu       sync.Mutex
	intents  map[string]stripe.PaymentIntent
	refunded []string
}

func (f *fakeStripe) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	f.mu.Lock()
	defer f.mu.Unlock()
	switch {
	case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/payment_intents/"):
		pi, ok := f.intents[strings.TrimPrefix(r.URL.Path, "/payment_intents/")]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error":{"code":"resource_missing"}}`))
			return
		}
		json.NewEncoder(w).Encode(pi)
	case r.Method == http.MethodPost && r.URL.Path == "/refunds":
		id := r.FormValue("payment_intent")
		pi := f.intents[id]
		pi.LatestCharge = &struct {
			AmountRefunded int `json:"amount_refunded"`
		}{pi.Amount}
		f.intents[id] = pi
		f.refunded = append(f.refunded, id)
		w.Write([]byte(`{}`))
	default:
		w.WriteHeader(http.StatusNotFound)
	}
}

// TestPayAndCancel checks booking from a payment: it's idempotent, checks whose payment it is, refunds when the seats
// are gone or the price is wrong, never books a refunded payment, and that cancelling refunds and frees the seats only
// for the owner, up to an hour before the showing. Needs DATABASE_URL. The booking code commits, so rows are deleted after.
func TestPayAndCancel(t *testing.T) {
	if os.Getenv("DATABASE_URL") == "" {
		t.Skip("DATABASE_URL not set")
	}
	ctx := context.Background()
	db.Connect(ctx)
	db.Migrate()
	t.Setenv("REDIS_ADDRESS", "127.0.0.1:1") // holds are skipped
	rdb.Connect(ctx)
	t.Setenv("STRIPE_SECRET_KEY", "sk_test")
	fake := &fakeStripe{intents: map[string]stripe.PaymentIntent{}}
	srv := httptest.NewServer(fake)
	defer srv.Close()
	stripe.API = srv.URL

	one := func(sql string, args ...any) string {
		t.Helper()
		var id string
		if err := db.Pool.QueryRow(ctx, sql, args...).Scan(&id); err != nil {
			t.Fatalf("%s: %v", sql, err)
		}
		return id
	}
	alice := one(`INSERT INTO users (first_name, last_name, email) VALUES ('A', 'A', 'alice@pay.test') RETURNING user_id`)
	bob := one(`INSERT INTO users (first_name, last_name, email) VALUES ('B', 'B', 'bob@pay.test') RETURNING user_id`)
	movie := one(`INSERT INTO movies (title, genre, runtime_minutes, rating) VALUES ('Pay test', 'Drama', 90, 'M') RETURNING movie_id`)
	room := one(`WITH c AS (INSERT INTO cinemas (name, address, city, state, country, postal_code, lat, lng)
		VALUES ('Pay test', '1 Test St', 'Melbourne', 'VIC', 'AU', '3000', -37.81, 144.96) RETURNING cinema_id)
 INSERT INTO rooms (cinema_id, name, capacity) SELECT cinema_id, 'Pay test screen', 3 FROM c RETURNING room_id`)
	format := one(`INSERT INTO event_formats (name, base_price_cents) VALUES ('Pay test', 1500) RETURNING format_id`)
	t.Cleanup(func() {
		db.Pool.Exec(ctx, `DELETE FROM outbox WHERE key = ANY($1::text[])`, []string{alice, bob})
		db.Pool.Exec(ctx, `DELETE FROM users WHERE user_id = ANY($1::uuid[])`, []string{alice, bob}) // and their bookings
		db.Pool.Exec(ctx, `DELETE FROM events WHERE room_id = $1`, room)
		db.Pool.Exec(ctx, `DELETE FROM rooms WHERE room_id = $1`, room)
		db.Pool.Exec(ctx, `DELETE FROM cinemas WHERE name = 'Pay test'`)
		db.Pool.Exec(ctx, `DELETE FROM movies WHERE movie_id = $1`, movie)
		db.Pool.Exec(ctx, `DELETE FROM event_formats WHERE format_id = $1`, format)
	})
	var seats []string
	for _, n := range []int{1, 2, 3} {
		seats = append(seats, one(`INSERT INTO seats (room_id, row_label, seat_number) VALUES ($1, 'A', $2) RETURNING seat_id`, room, n))
	}
	event := func(in time.Duration) string {
		return one(`INSERT INTO events (starts_at, room_id, movie_id, format_id) VALUES ($1, $2, $3, $4) RETURNING event_id`,
			time.Now().Add(in), room, movie, format)
	}
	later, soon := event(3*time.Hour), event(30*time.Minute)
	pay := func(id, eventID, userID string, amount int, seatIDs ...string) {
		fake.intents[id] = stripe.PaymentIntent{ID: id, Amount: amount, Status: "succeeded", Metadata: map[string]string{
			"event_id": eventID, "user_id": userID, "seat_ids": strings.Join(seatIDs, ","),
		}}
	}
	status := func(err error) int {
		if e, ok := errors.AsType[*response.AppError](err); ok {
			return e.Status
		}
		return 0
	}

	pay("pi_ok", later, alice, 3000, seats[0], seats[1])
	b, err := confirm(ctx, "pi_ok", later, alice)
	if err != nil {
		t.Fatalf("confirm = %v", err)
	}
	if again, err := confirm(ctx, "pi_ok", "", ""); err != nil || again != b {
		t.Fatalf("webhook after page = %+v, %v; want the same booking %+v", again, err, b)
	}
	if got, _ := dbListUserBookings(ctx, db.Pool, alice, b.Ref); len(got) != 1 || got[0].BookingID != b.BookingID {
		t.Fatalf("lookup by ref %s = %+v", b.Ref, got)
	}
	if _, err := confirm(ctx, "pi_ok", later, bob); !errors.Is(err, errNoPayment) {
		t.Fatalf("confirm someone else's payment = %v, want errNoPayment", err)
	}
	if _, err := confirm(ctx, "pi_missing", later, alice); !errors.Is(err, errNoPayment) {
		t.Fatalf("confirm unknown payment = %v, want errNoPayment", err)
	}

	pay("pi_pending", later, bob, 1500, seats[2])
	p := fake.intents["pi_pending"]
	p.Status = "requires_payment_method"
	fake.intents["pi_pending"] = p
	if _, err := confirm(ctx, "pi_pending", later, bob); !errors.Is(err, errNotPaid) {
		t.Fatalf("confirm unpaid = %v, want errNotPaid", err)
	}

	pay("pi_clash", later, bob, 3000, seats[1], seats[2])
	if _, err := confirm(ctx, "pi_clash", later, bob); status(err) != http.StatusConflict || !strings.Contains(err.Error(), "refunded") {
		t.Fatalf("confirm taken seat = %v, want a 409 that says it's refunded", err)
	}
	if _, err := confirm(ctx, "pi_clash", "", ""); !errors.Is(err, errRefunded) {
		t.Fatalf("webhook retry of a refunded payment = %v, want errRefunded", err)
	}

	pay("pi_cheap", later, bob, 100, seats[2])
	if _, err := confirm(ctx, "pi_cheap", later, bob); status(err) != http.StatusConflict {
		t.Fatalf("confirm wrong amount = %v, want 409", err)
	}
	if want := []string{"pi_clash", "pi_cheap"}; strings.Join(fake.refunded, ",") != strings.Join(want, ",") {
		t.Fatalf("refunded %v, want %v", fake.refunded, want)
	}

	if err := cancel(ctx, b.BookingID, bob); !errors.Is(err, errNoBooking) {
		t.Fatalf("cancel someone else's = %v, want errNoBooking", err)
	}
	if err := cancel(ctx, "not-a-uuid", alice); !errors.Is(err, errNoBooking) {
		t.Fatalf("cancel bad ID = %v, want errNoBooking", err)
	}
	if err := cancel(ctx, b.BookingID, alice); err != nil {
		t.Fatalf("cancel = %v", err)
	}
	if fake.refunded[len(fake.refunded)-1] != "pi_ok" {
		t.Fatalf("cancel didn't refund pi_ok: %v", fake.refunded)
	}
	if f, err := checkSeats(ctx, db.Pool, later, seats[:2]); err != nil || f.Free != 2 {
		t.Fatalf("seats after cancel = %+v, %v; want both free", f, err)
	}
	if _, err := confirm(ctx, "pi_ok", "", ""); !errors.Is(err, errRefunded) {
		t.Fatalf("webhook redelivered after cancel = %v, want errRefunded", err)
	}

	pay("pi_soon", soon, alice, 1500, seats[0])
	s, err := confirm(ctx, "pi_soon", soon, alice)
	if err != nil {
		t.Fatalf("confirm soon = %v", err)
	}
	if err := cancel(ctx, s.BookingID, alice); !errors.Is(err, errTooLate) {
		t.Fatalf("cancel within the hour = %v, want errTooLate", err)
	}
}
