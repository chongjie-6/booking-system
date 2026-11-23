// Shapes returned by GET /movies and GET /events/{id}/seats
export type Showing = { eventID: string; startsAt: string; priceCents: number; room: string }

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
  taken: boolean
}

export type Showtime = { id: string; movie: Movie; start: Date; priceCents: number; room: string }

export type Booking = { ref: string; name: string; email: string; showtime: Showtime; seats: Seat[] }

// Shape returned by GET /bookings, latest showing first
export type UserBooking = {
  bookingID: string
  ref: string
  eventID: string
  title: string
  startsAt: string
  room: string
  priceCents: number
  seats: SeatPlace[]
}

// What a ticket needs, so it can show a booking from the API as well as a showtime being booked
export type SeatPlace = Pick<Seat, 'row' | 'number'>
export type TicketShowtime = Pick<Showtime, 'start' | 'priceCents' | 'room'> & { movie: Pick<Movie, 'title'> }

export const MAX_SEATS = 10

export function upcomingDays(count = 3): Date[] {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return Array.from({ length: count }, (_, i) => new Date(today.getFullYear(), today.getMonth(), today.getDate() + i))
}

const toShowtime = (movie: Movie, s: Showing): Showtime =>
  ({ id: s.eventID, movie, start: new Date(s.startsAt), priceCents: s.priceCents, room: s.room })

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

export const seatLabel = (s: SeatPlace) => `${s.row}${s.number}`
export const total = (showtime: Pick<Showtime, 'priceCents'>, seats: number) => money.format((showtime.priceCents * seats) / 100)

export const money = new Intl.NumberFormat('en-AU', { style: 'currency', currency: 'AUD' })
export const formatRuntime = (minutes: number) => `${Math.floor(minutes / 60)}h ${minutes % 60}m`
export const formatTime = (d: Date) => d.toLocaleTimeString('en-AU', { hour: 'numeric', minute: '2-digit' })
export const formatDay = (d: Date) => d.toLocaleDateString('en-AU', { weekday: 'short', day: 'numeric', month: 'short' })
