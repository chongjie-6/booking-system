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

// Showing is one screening of a movie (an events row). Format is e.g. "IMAX"; Room is the room's name, e.g. "Screen 1";
// CinemaID and Cinema are the site the room is in, for searching sessions by cinema.
type Showing struct {
	EventID  string    `json:"eventID"`
	StartsAt time.Time `json:"startsAt"`
	Format   string    `json:"format"`
	Room     string    `json:"room"`
	CinemaID string    `json:"cinemaID"`
	Cinema   string    `json:"cinema"`
}

// Cinema is one site, e.g. "Melbourne Central", with its street address and WGS84 position, for sorting by distance.
type Cinema struct {
	CinemaID string  `db:"cinema_id" json:"cinemaID"`
	Name     string  `db:"name" json:"name"`
	Address  string  `db:"address" json:"address"`
	City     string  `db:"city" json:"city"`
	State    string  `db:"state" json:"state"`
	Lat      float64 `db:"lat" json:"lat"`
	Lng      float64 `db:"lng" json:"lng"`
}

// Seat is one seat in a showing's room. PriceCents is the format's price plus the seat type's surcharge.
// Taken means it's already booked for that showing.
type Seat struct {
	SeatID     string `db:"seat_id" json:"seatID"`
	Row        string `db:"row_label" json:"row"`
	Number     int    `db:"seat_number" json:"number"`
	Type       string `db:"seat_type" json:"type"`
	PriceCents int    `db:"price_cents" json:"priceCents"`
	Taken      bool   `db:"taken" json:"taken"`
}
