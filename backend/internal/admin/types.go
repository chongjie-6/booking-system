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

// eventRequest is a showing to add or replace. Stripe's smallest charge in AUD is 50 cents.
type eventRequest struct {
	MovieID    string    `json:"movieID" validate:"required,uuid"`
	RoomID     string    `json:"roomID" validate:"required,uuid"`
	StartsAt   time.Time `json:"startsAt" validate:"required"`
	PriceCents int       `json:"priceCents" validate:"min=50,max=100000"`
}

// created is the ID of a new film or showing.
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
	EventID    string    `json:"eventID"`
	StartsAt   time.Time `json:"startsAt"`
	PriceCents int       `json:"priceCents"`
	RoomID     string    `json:"roomID"`
	Room       string    `json:"room"`
	Booked     int       `json:"booked"`
}

// Room is a screen showings can be put in.
type Room struct {
	RoomID   string `db:"room_id" json:"roomID"`
	Name     string `db:"name" json:"name"`
	Capacity int    `db:"capacity" json:"capacity"`
}
