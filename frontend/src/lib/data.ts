// Shapes returned by GET /movies and GET /events/{id}/seats
export type Showing = { eventID: string; startsAt: string; format: string; room: string } // format e.g. 'IMAX'

export type Movie = {
  movieID: string
  title: string
  genre: string
  runtimeMinutes: number
  rating: string
  synopsis: string
  featured: boolean
  posterPath: string // TMDB image path, '' when there's none
  backdropPath: string // likewise
  trailerKey: string // YouTube video key, '' when there's none
  showings: Showing[]
}

export type Seat = {
  seatID: string
  row: string
  number: number
  type: 'standard' | 'wheelchair' | 'companion' | 'premium'
  priceCents: number // the showing's format price plus the seat type's surcharge
  taken: boolean
}

export type Showtime = { id: string; movie: Movie; start: Date; format: string; room: string }

export type Booking = { ref: string; name: string; email: string; showtime: Showtime; seats: Seat[] }

// Shape returned by GET /bookings, latest showing first
export type UserBooking = {
  bookingID: string
  ref: string
  eventID: string
  title: string
  startsAt: string
  room: string
  totalCents: number // what the seats sold for
  seats: SeatPlace[]
}

// Shapes returned by GET /admin/movies and GET /admin/rooms. booked counts sold seats; a showing with any is locked.
export type AdminShowing = {
  eventID: string
  startsAt: string
  formatID: string
  format: string
  roomID: string
  room: string
  booked: number
}
export type AdminMovie = Pick<Movie, 'movieID' | 'title' | 'genre' | 'runtimeMinutes' | 'rating' | 'synopsis' | 'featured'> & {
  tmdbID: number | null
  showings: AdminShowing[]
}
export type Room = { roomID: string; name: string; capacity: number }
export type Format = { formatID: string; name: string; basePriceCents: number }
export type Surcharge = { seatType: Seat['type']; surchargeCents: number }

// What a ticket needs, so it can show a booking from the API as well as a showtime being booked
export type SeatPlace = Pick<Seat, 'row' | 'number'>
export type TicketShowtime = Pick<Showtime, 'start' | 'room'> & { movie: Pick<Movie, 'title'> }

export const MAX_SEATS = 10

export function upcomingDays(count = 3): Date[] {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return Array.from({ length: count }, (_, i) => new Date(today.getFullYear(), today.getMonth(), today.getDate() + i))
}

const toShowtime = (movie: Movie, s: Showing): Showtime =>
  ({ id: s.eventID, movie, start: new Date(s.startsAt), format: s.format, room: s.room })

export function showtimesFor(movie: Movie, day: Date): Showtime[] {
  return movie.showings.map((s) => toShowtime(movie, s)).filter((s) => s.start.toDateString() === day.toDateString())
}

export function findShowtime(movies: Movie[], eventID: string): Showtime | undefined {
  for (const movie of movies) {
    const s = movie.showings.find((s) => s.eventID === eventID)
    if (s) return toShowtime(movie, s)
  }
}

export const posterURL = (path: string) => `https://image.tmdb.org/t/p/w342${path}`
export const backdropURL = (path: string) => `https://image.tmdb.org/t/p/w1280${path}`

// Films without a TMDB poster get a drawn one, in a stable colour from the film's ID.
export function hue(movieID: string): number {
  let h = 0
  for (const ch of movieID) h = (h * 31 + ch.charCodeAt(0)) % 360
  return h
}

export const slug = (title: string) => title.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/^-|-$/g, '')

// The API sends the latest showing first; upcoming ones read better soonest first.
export function splitBookings(bookings: UserBooking[], now: number) {
  const upcoming = bookings.filter((b) => Date.parse(b.startsAt) > now).reverse()
  const past = bookings.filter((b) => Date.parse(b.startsAt) <= now)
  return { upcoming, past }
}

export const ticketShowtime = (b: UserBooking): TicketShowtime => ({
  movie: { title: b.title },
  start: new Date(b.startsAt),
  room: b.room,
})

// Matches the backend: bookings can be cancelled and refunded up to an hour before the showing.
export const canCancel = (b: Pick<UserBooking, 'startsAt'>, now: number) => Date.parse(b.startsAt) - now > 60 * 60 * 1000

export const seatLabel = (s: SeatPlace) => `${s.row}${s.number}`
export const totalCents = (seats: Pick<Seat, 'priceCents'>[]) => seats.reduce((t, s) => t + s.priceCents, 0)
export const total = (seats: Pick<Seat, 'priceCents'>[]) => money.format(totalCents(seats) / 100)

// An ISO time as a datetime-local input's value, in the browser's time zone, e.g. 2026-11-24T19:30.
export function toLocalInput(iso: string) {
  const d = new Date(iso)
  return new Date(d.getTime() - d.getTimezoneOffset() * 60_000).toISOString().slice(0, 16)
}

export const money = new Intl.NumberFormat('en-AU', { style: 'currency', currency: 'AUD' })
export const formatRuntime = (minutes: number) => `${Math.floor(minutes / 60)}h ${minutes % 60}m`
export const formatTime = (d: Date) => d.toLocaleTimeString('en-AU', { hour: 'numeric', minute: '2-digit' })
export const formatDay = (d: Date) => d.toLocaleDateString('en-AU', { weekday: 'short', day: 'numeric', month: 'short' })
