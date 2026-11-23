package booking

import "time"

// seatsRequest names the seats to hold, release or book for the showing in the path. 10 matches the frontend's MAX_SEATS.
type seatsRequest struct {
	SeatIDs []string `json:"seatIDs" validate:"required,min=1,max=10,unique,dive,uuid"`
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

// freeSeats is how many of the asked-for seats a showing can still sell, and the showing's room.
type freeSeats struct {
	RoomID string `db:"room_id"`
	Free   int    `db:"free"`
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
