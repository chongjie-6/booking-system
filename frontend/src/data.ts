export type Movie = {
  id: string
  title: string
  genre: string
  runtime: number
  rating: string
  synopsis: string
  hue: number
  screen: number
  times: string[]
}

export type Showtime = { id: string; movie: Movie; start: Date }

export type Booking = { ref: string; name: string; email: string; showtime: Showtime; seats: string[] }

export const ROWS = 'ABCDEFGH'.split('')
export const SEATS_PER_ROW = 14
export const AISLES_AFTER = [3, 11]
export const MAX_SEATS = 10
export const PRICE = 16.5

export const MOVIES: Movie[] = [
  {
    id: 'long-quiet',
    title: 'The Long Quiet',
    genre: 'Drama',
    runtime: 128,
    rating: 'M',
    synopsis: 'A lighthouse keeper sees out her last winter on an island the ferry no longer visits.',
    hue: 205,
    screen: 1,
    times: ['13:15', '16:40', '20:10'],
  },
  {
    id: 'kite-season',
    title: 'Kite Season',
    genre: 'Comedy',
    runtime: 97,
    rating: 'PG',
    synopsis: 'Two rival kite makers are forced to share one stall at the county fair.',
    hue: 42,
    screen: 2,
    times: ['10:30', '14:00', '18:20'],
  },
  {
    id: 'reservoir',
    title: 'The Reservoir',
    genre: 'Thriller',
    runtime: 112,
    rating: 'MA15+',
    synopsis: 'A salvage diver finds a sealed car at the bottom of the reservoir. Someone wants it left there.',
    hue: 160,
    screen: 3,
    times: ['17:45', '21:30'],
  },
  {
    id: 'saltwater-kids',
    title: 'Saltwater Kids',
    genre: 'Animation',
    runtime: 88,
    rating: 'G',
    synopsis: 'A crab who hates getting wet joins the reef swimming team.',
    hue: 15,
    screen: 2,
    times: ['09:45', '12:00', '15:15'],
  },
  {
    id: 'last-train-north',
    title: 'Last Train North',
    genre: 'Science fiction',
    runtime: 141,
    rating: 'M',
    synopsis: 'The final train out of a frozen city is carrying one passenger too many.',
    hue: 265,
    screen: 1,
    times: ['19:00', '22:15'],
  },
]

export function upcomingDays(count = 3): Date[] {
  const today = new Date()
  today.setHours(0, 0, 0, 0)
  return Array.from({ length: count }, (_, i) => new Date(today.getFullYear(), today.getMonth(), today.getDate() + i))
}

export function showtimesFor(movie: Movie, day: Date): Showtime[] {
  return movie.times.map((t) => {
    const [h, m] = t.split(':').map(Number)
    const start = new Date(day)
    start.setHours(h, m)
    return { id: `${movie.id}-${day.toDateString()}-${t}`, movie, start }
  })
}

// Seeded from the showtime id so the same session always shows the same taken seats.
export function takenSeats(showtimeId: string): Set<string> {
  let seed = 0
  for (const ch of showtimeId) seed = (seed * 31 + ch.charCodeAt(0)) | 0
  const random = () => {
    seed = (seed + 0x6d2b79f5) | 0
    let t = Math.imul(seed ^ (seed >>> 15), 1 | seed)
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296
  }
  const taken = new Set<string>()
  for (const row of ROWS)
    for (let n = 1; n <= SEATS_PER_ROW; n++) if (random() < 0.3) taken.add(`${row}${n}`)
  return taken
}

export const money = new Intl.NumberFormat('en-AU', { style: 'currency', currency: 'AUD' })
export const formatTime = (d: Date) => d.toLocaleTimeString('en-AU', { hour: 'numeric', minute: '2-digit' })
export const formatDay = (d: Date) => d.toLocaleDateString('en-AU', { weekday: 'short', day: 'numeric', month: 'short' })
