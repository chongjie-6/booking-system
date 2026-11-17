// Shapes returned by GET /movies and GET /events/{id}/seats
export type Showing = { eventID: string; startsAt: string; priceCents: number; room: string }

export type Movie = {
  movieID: string
  title: string
  genre: string
  runtimeMinutes: number
  rating: string
  synopsis: string
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

export const MAX_SEATS = 10

export function upcomingDays(count = 3): Date[] {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return Array.from({ length: count }, (_, i) => new Date(today.getFullYear(), today.getMonth(), today.getDate() + i))
}

export function showtimesFor(movie: Movie, day: Date): Showtime[] {
  return movie.showings
    .map((s) => ({ id: s.eventID, movie, start: new Date(s.startsAt), priceCents: s.priceCents, room: s.room }))
    .filter((s) => s.start.toDateString() === day.toDateString())
}

// Posters are drawn, not images, so each film gets a stable colour from its ID.
export function hue(movieID: string): number {
  let h = 0
  for (const ch of movieID) h = (h * 31 + ch.charCodeAt(0)) % 360
  return h
}

export const seatLabel = (s: Seat) => `${s.row}${s.number}`
export const total = (showtime: Showtime, seats: number) => money.format((showtime.priceCents * seats) / 100)

export const money = new Intl.NumberFormat('en-AU', { style: 'currency', currency: 'AUD' })
export const formatTime = (d: Date) => d.toLocaleTimeString('en-AU', { hour: 'numeric', minute: '2-digit' })
export const formatDay = (d: Date) => d.toLocaleDateString('en-AU', { weekday: 'short', day: 'numeric', month: 'short' })
