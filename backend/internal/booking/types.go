package booking

import "time"

// seatsRequest names the seats to hold, release or book for the showing in the path. 10 matches the frontend's MAX_SEATS.
type seatsRequest struct {
	SeatIDs []string `json:"seatIDs" validate:"required,min=1,max=10,unique,dive,uuid"`
}

// bookRequest names the Stripe payment to book seats with; the seats come from the payment's metadata.
type bookRequest struct {
	PaymentIntentID string `json:"paymentIntentID" validate:"required,startswith=pi_,max=255"`
}

// paymentResponse is what the checkout page needs to show Stripe's card form and take the payment.
type paymentResponse struct {
	ClientSecret   string `json:"clientSecret"`
	PublishableKey string `json:"publishableKey"`
	AmountCents    int    `json:"amountCents"`
}

// holdResponse is how long the seats stay held. A duration rather than a time, so the client's clock doesn't matter.
type holdResponse struct {
	HoldSeconds int `json:"holdSeconds"`
}

// Booking is a confirmed booking. Ref is the short code shown on the ticket and in the email.
type Booking struct {
	BookingID string `json:"bookingID"`
	Ref       string `json:"ref"`
}

// freeSeats is how many of the asked-for seats a showing can still sell, with the showing's room and seat price.
type freeSeats struct {
	RoomID     string `db:"room_id"`
	PriceCents int    `db:"price_cents"`
	Free       int    `db:"free"`
}

// lockedBooking is a booking being cancelled: its payment (nil if it was made before payments) and whether the
// showing is still far enough off to cancel.
type lockedBooking struct {
	PaymentIntentID *string `db:"payment_intent_id"`
	Cancellable     bool    `db:"cancellable"`
}

// UserBooking is one of the signed-in user's bookings (GET /bookings), with what its ticket shows.
type UserBooking struct {
	BookingID  string       `db:"booking_id" json:"bookingID"`
	Ref        string       `db:"ref" json:"ref"`
	EventID    string       `db:"event_id" json:"eventID"`
	Title      string       `db:"title" json:"title"`
	StartsAt   time.Time    `db:"starts_at" json:"startsAt"`
	Room       string       `db:"room" json:"room"`
	PriceCents int          `db:"price_cents" json:"priceCents"`
	Seats      []bookedSeat `db:"seats" json:"seats"`
}

// bookedSeat is a seat on a UserBooking, e.g. row "C", number 4.
type bookedSeat struct {
	Row    string `json:"row"`
	Number int    `json:"number"`
}
