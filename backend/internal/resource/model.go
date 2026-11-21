package resource

import "time"

// Movie is a film with its upcoming showings, soonest first.
type Movie struct {
	MovieID        string    `db:"movie_id" json:"movieID"`
	Title          string    `db:"title" json:"title"`
	Genre          string    `db:"genre" json:"genre"`
	RuntimeMinutes int       `db:"runtime_minutes" json:"runtimeMinutes"`
	Rating         string    `db:"rating" json:"rating"`
	Synopsis       string    `db:"synopsis" json:"synopsis"`
	Featured       bool      `db:"featured" json:"featured"`          // pinned to the top of the listing
	PosterPath     string    `db:"poster_path" json:"posterPath"`     // TMDB poster, e.g. /abc.jpg; "" until fetched or if there is none
	BackdropPath   string    `db:"backdrop_path" json:"backdropPath"` // TMDB wide still, like PosterPath
	TrailerKey     string    `db:"trailer_key" json:"trailerKey"`     // YouTube video key; "" until fetched or if there is none
	Showings       []Showing `db:"showings" json:"showings"`
}

// Showing is one screening of a movie (an events row). Room is the room's name, e.g. "Screen 1".
type Showing struct {
	EventID    string    `json:"eventID"`
	StartsAt   time.Time `json:"startsAt"`
	PriceCents int       `json:"priceCents"`
	Room       string    `json:"room"`
}

// Seat is one seat in a showing's room. Taken means it's already booked for that showing.
type Seat struct {
	SeatID string `db:"seat_id" json:"seatID"`
	Row    string `db:"row_label" json:"row"`
	Number int    `db:"seat_number" json:"number"`
	Type   string `db:"seat_type" json:"type"`
	Taken  bool   `db:"taken" json:"taken"`
}
