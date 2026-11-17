package booking

// seatsRequest names the seats to hold, release or book for the showing in the path. 10 matches the frontend's MAX_SEATS.
type seatsRequest struct {
	SeatIDs []string `json:"seatIDs" validate:"required,min=1,max=10,unique,dive,uuid"`
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
