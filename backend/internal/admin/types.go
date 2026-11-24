package admin

import "time"

// movieRequest is a film to add or replace. A new or changed TMDBID refetches its poster, trailer, synopsis and runtime.
type movieRequest struct {
	Title          string `json:"title" validate:"required,max=200"`
	Genre          string `json:"genre" validate:"required,max=100"`
	RuntimeMinutes int    `json:"runtimeMinutes" validate:"required,min=1,max=600"`
	Rating         string `json:"rating" validate:"required,max=10"`
	Synopsis       string `json:"synopsis" validate:"max=2000"`
	TMDBID         *int   `json:"tmdbID" validate:"omitempty,min=1"`
	Featured       bool   `json:"featured"`
}

// eventRequest is a showing to add or replace. Its seats' price comes from the format.
type eventRequest struct {
	MovieID  string    `json:"movieID" validate:"required,uuid"`
	RoomID   string    `json:"roomID" validate:"required,uuid"`
	StartsAt time.Time `json:"startsAt" validate:"required"`
	FormatID string    `json:"formatID" validate:"required,uuid"`
}

// formatRequest is a showing format to add or replace. Stripe's smallest charge in AUD is 50 cents.
type formatRequest struct {
	Name           string `json:"name" validate:"required,max=50"`
	BasePriceCents int    `json:"basePriceCents" validate:"min=50,max=100000"`
}

// surchargeRequest is what a seat type adds to the format's price.
type surchargeRequest struct {
	SurchargeCents int `json:"surchargeCents" validate:"min=0,max=100000"`
}

// created is the ID of a new film, showing or format.
type created struct {
	ID string `json:"id"`
}

// Movie is a film as staff see it, with every upcoming showing.
type Movie struct {
	MovieID        string    `db:"movie_id" json:"movieID"`
	Title          string    `db:"title" json:"title"`
	Genre          string    `db:"genre" json:"genre"`
	RuntimeMinutes int       `db:"runtime_minutes" json:"runtimeMinutes"`
	Rating         string    `db:"rating" json:"rating"`
	Synopsis       string    `db:"synopsis" json:"synopsis"`
	TMDBID         *int      `db:"tmdb_id" json:"tmdbID"`
	Featured       bool      `db:"featured" json:"featured"`
	Showings       []Showing `db:"showings" json:"showings"`
}

// Showing is an upcoming showing. Booked is how many seats are sold; a showing with any can't be changed.
type Showing struct {
	EventID  string    `json:"eventID"`
	StartsAt time.Time `json:"startsAt"`
	FormatID string    `json:"formatID"`
	Format   string    `json:"format"`
	RoomID   string    `json:"roomID"`
	Room     string    `json:"room"`
	Booked   int       `json:"booked"`
}

// Format is a kind of showing, e.g. "IMAX", with what a standard seat at one costs.
type Format struct {
	FormatID       string `db:"format_id" json:"formatID"`
	Name           string `db:"name" json:"name"`
	BasePriceCents int    `db:"base_price_cents" json:"basePriceCents"`
}

// Surcharge is what a seat type, e.g. "premium", adds to the price of a seat at any showing.
type Surcharge struct {
	SeatType       string `db:"seat_type" json:"seatType"`
	SurchargeCents int    `db:"surcharge_cents" json:"surchargeCents"`
}

// Room is a screen showings can be put in.
type Room struct {
	RoomID   string `db:"room_id" json:"roomID"`
	Name     string `db:"name" json:"name"`
	Capacity int    `db:"capacity" json:"capacity"`
}
