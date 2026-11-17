import { useEffect, useState, type CSSProperties, type Dispatch, type SetStateAction } from 'react'
import { api, type User } from './api'
import { Auth } from './Auth'
import {
  MAX_SEATS,
  formatDay,
  formatTime,
  hue,
  seatLabel,
  showtimesFor,
  total,
  upcomingDays,
  type Booking,
  type Movie,
  type Seat,
  type Showtime,
} from './data'

type View =
  | { step: 'films' }
  | { step: 'seats'; showtime: Showtime }
  | { step: 'checkout'; showtime: Showtime }
  | { step: 'done'; booking: Booking }
  | { step: 'signin'; resetToken?: string }

// A reset email links to /#reset=<token>. Read it once, then drop it from the address bar and history.
const resetToken = new URLSearchParams(location.hash.slice(1)).get('reset') ?? undefined
if (resetToken) history.replaceState(null, '', location.pathname + location.search)

export default function App() {
  const [view, setView] = useState<View>(resetToken ? { step: 'signin', resetToken } : { step: 'films' })
  const [day, setDay] = useState(0)
  const [seats, setSeats] = useState<Seat[]>([])
  // seat IDs booked in this tab, per showing
  const [booked, setBooked] = useState<Record<string, string[]>>({})
  // undefined while the first lookup runs, null when signed out
  const [user, setUser] = useState<User | null>()

  useEffect(() => {
    api<User>('/user').then(setUser, () => setUser(null))
  }, [])

  function go(next: View) {
    setView(next)
    window.scrollTo(0, 0)
  }

  function signOut() {
    api('/auth/logout', { method: 'POST' }).then(
      () => setUser(null),
      () => alert("Couldn't sign you out. Please try again."),
    )
  }

  // ponytail: bookings stay in memory until the booking endpoints exist
  function confirm(showtime: Showtime, user: User) {
    const ref = crypto.randomUUID().slice(0, 6).toUpperCase()
    const booking = { ref, name: user.firstName, email: user.email, showtime, seats }
    setBooked({ ...booked, [showtime.id]: [...(booked[showtime.id] ?? []), ...seats.map((s) => s.seatID)] })
    setSeats([])
    go({ step: 'done', booking })
  }

  return (
    <>
      <header className="masthead">
        <button className="wordmark" onClick={() => go({ step: 'films' })}>
          Lantern
        </button>
        {user ? (
          <div className="account">
            <span>{user.firstName}</span>
            <button className="link" onClick={signOut}>
              Sign out
            </button>
          </div>
        ) : (
          user === null &&
          view.step !== 'signin' && (
            <button className="link" onClick={() => go({ step: 'signin' })}>
              Sign in
            </button>
          )
        )}
      </header>

      {view.step === 'signin' && (
        <main>
          <Auth
            mode={view.resetToken ? 'reset' : 'login'}
            resetToken={view.resetToken}
            onSignedIn={(u) => {
              setUser(u)
              go({ step: 'films' })
            }}
          />
        </main>
      )}

      {view.step === 'films' && (
        <Films
          day={day}
          onDay={setDay}
          onPick={(showtime) => {
            setSeats([])
            go({ step: 'seats', showtime })
          }}
        />
      )}
      {view.step === 'seats' && (
        <SeatPicker
          showtime={view.showtime}
          booked={booked[view.showtime.id] ?? []}
          seats={seats}
          onSeats={setSeats}
          onBack={() => go({ step: 'films' })}
          onContinue={() => go({ step: 'checkout', showtime: view.showtime })}
        />
      )}
      {view.step === 'checkout' && (
        <Checkout
          showtime={view.showtime}
          seats={seats}
          user={user}
          onSignedIn={setUser}
          onBack={() => go({ step: 'seats', showtime: view.showtime })}
          onConfirm={(u) => confirm(view.showtime, u)}
        />
      )}
      {view.step === 'done' && <Done booking={view.booking} onAgain={() => go({ step: 'films' })} />}

      <footer className="credit">
        <a href="https://db-ip.com">IP Geolocation by DB-IP</a>
      </footer>
    </>
  )
}

function Films({ day, onDay, onPick }: { day: number; onDay: (i: number) => void; onPick: (s: Showtime) => void }) {
  const days = upcomingDays()
  const now = new Date()
  // undefined while loading, null if it failed
  const [movies, setMovies] = useState<Movie[] | null>()

  useEffect(() => {
    api<Movie[]>('/movies').then(setMovies, () => setMovies(null))
  }, [])

  return (
    <main>
      <h1>Now showing</h1>
      {movies === null && (
        <p className="error" role="alert">
          Couldn't load the films. Please refresh to try again.
        </p>
      )}
      {movies?.length === 0 && <p className="none">Nothing is showing in the next few days.</p>}
      <nav className="days" aria-label="Day">
        {days.map((d, i) => (
          <button key={i} aria-pressed={i === day} onClick={() => onDay(i)}>
            {i === 0 ? 'Today' : i === 1 ? 'Tomorrow' : formatDay(d)}
          </button>
        ))}
      </nav>

      <ul className="listing">
        {movies?.map((movie) => {
          const sessions = showtimesFor(movie, days[day]).filter((s) => s.start > now)
          return (
            <li key={movie.movieID} className="film">
              <div className="poster" style={{ '--hue': hue(movie.movieID) } as CSSProperties} aria-hidden="true">
                <span>{movie.title}</span>
              </div>
              <div>
                <h2>{movie.title}</h2>
                <p className="meta">
                  <span>{movie.genre}</span>
                  <span>
                    {Math.floor(movie.runtimeMinutes / 60)}h {movie.runtimeMinutes % 60}m
                  </span>
                  <span className="rating">{movie.rating}</span>
                </p>
                <p className="synopsis">{movie.synopsis}</p>
                {sessions.length ? (
                  <div className="times">
                    {sessions.map((s) => (
                      <button key={s.id} aria-label={`${movie.title} at ${formatTime(s.start)}`} onClick={() => onPick(s)}>
                        {formatTime(s.start)}
                      </button>
                    ))}
                  </div>
                ) : (
                  <p className="none">No more sessions today. Try tomorrow.</p>
                )}
              </div>
            </li>
          )
        })}
      </ul>
    </main>
  )
}

function SeatPicker({
  showtime,
  booked,
  seats,
  onSeats,
  onBack,
  onContinue,
}: {
  showtime: Showtime
  booked: string[]
  seats: Seat[]
  onSeats: Dispatch<SetStateAction<Seat[]>>
  onBack: () => void
  onContinue: () => void
}) {
  // undefined while loading, null if it failed
  const [all, setAll] = useState<Seat[] | null>()
  const atMax = seats.length >= MAX_SEATS

  useEffect(() => {
    api<Seat[]>(`/events/${showtime.id}/seats`).then(setAll, () => setAll(null))
  }, [showtime.id])

  // The API sends seats ordered by row then number.
  const rows: Seat[][] = []
  for (const seat of all ?? []) {
    if (rows.at(-1)?.[0].row === seat.row) rows.at(-1)!.push(seat)
    else rows.push([seat])
  }

  function toggle(seat: Seat) {
    onSeats((seats) => {
      if (seats.some((s) => s.seatID === seat.seatID)) return seats.filter((s) => s.seatID !== seat.seatID)
      if (seats.length >= MAX_SEATS) return seats
      return [...seats, seat].sort((a, b) => a.row.localeCompare(b.row) || a.number - b.number)
    })
  }

  return (
    <main>
      <button className="back" onClick={onBack}>
        Back to films
      </button>
      <h1>{showtime.movie.title}</h1>
      <p className="meta">
        <span>
          {formatDay(showtime.start)}, {formatTime(showtime.start)}
        </span>
        <span>{showtime.room}</span>
      </p>

      {all === null && (
        <p className="error" role="alert">
          Couldn't load the seats. Go back and pick the session again.
        </p>
      )}
      <div className="auditorium">
        <div className="screen" aria-hidden="true" />
        <div className="seats" role="group" aria-label="Seats">
          {rows.map((row) => (
            <div className="row" key={row[0].row}>
              <span className="row-label" aria-hidden="true">
                {row[0].row}
              </span>
              {row.map((seat) => {
                const isTaken = seat.taken || booked.includes(seat.seatID)
                const selected = seats.some((s) => s.seatID === seat.seatID)
                return (
                  <button
                    key={seat.seatID}
                    className="seat"
                    disabled={isTaken}
                    aria-pressed={selected}
                    aria-label={`Row ${seat.row}, seat ${seat.number}${seat.type === 'standard' ? '' : `, ${seat.type}`}${isTaken ? ', taken' : ''}`}
                    onClick={() => toggle(seat)}
                  >
                    {selected && seat.number}
                  </button>
                )
              })}
              <span className="row-label" aria-hidden="true">
                {row[0].row}
              </span>
            </div>
          ))}
        </div>
      </div>

      <ul className="legend">
        <li>
          <span className="swatch" /> Available
        </li>
        <li>
          <span className="swatch selected" /> Your seats
        </li>
        <li>
          <span className="swatch taken" /> Taken
        </li>
      </ul>

      <footer className="tray">
        <div aria-live="polite">
          <p>
            {seats.length
              ? `${seats.length} ${seats.length === 1 ? 'seat' : 'seats'}: ${seats.map(seatLabel).join(', ')}`
              : `Choose up to ${MAX_SEATS} seats.`}
          </p>
          {atMax && <p className="hint">That's the most seats one booking can hold.</p>}
        </div>
        <strong className="total">{total(showtime, seats.length)}</strong>
        <button className="primary" disabled={!seats.length} onClick={onContinue}>
          Continue to checkout
        </button>
      </footer>
    </main>
  )
}

function Checkout({
  showtime,
  seats,
  user,
  onSignedIn,
  onBack,
  onConfirm,
}: {
  showtime: Showtime
  seats: Seat[]
  user: User | null | undefined
  onSignedIn: (u: User) => void
  onBack: () => void
  onConfirm: (u: User) => void
}) {
  return (
    <main>
      <button className="back" onClick={onBack}>
        Change seats
      </button>
      <h1>Confirm your booking</h1>
      <div className="checkout">
        {user ? (
          <div className="fields">
            <p>
              Booking as {user.firstName} {user.lastName}. We'll send your tickets to {user.email}.
            </p>
            <button className="primary" onClick={() => onConfirm(user)}>
              Confirm booking
            </button>
          </div>
        ) : user === null ? (
          <Auth onSignedIn={onSignedIn} />
        ) : (
          <div />
        )}
        <Ticket showtime={showtime} seats={seats} />
      </div>
    </main>
  )
}

function Done({ booking, onAgain }: { booking: Booking; onAgain: () => void }) {
  return (
    <main className="done">
      <h1>Booking confirmed</h1>
      <p>
        {booking.name}, your tickets are on their way to {booking.email}. Show reference {booking.ref} at the door.
      </p>
      <Ticket showtime={booking.showtime} seats={booking.seats} bookingRef={booking.ref} />
      <button className="primary" onClick={onAgain}>
        Book another film
      </button>
    </main>
  )
}

function Ticket({ showtime, seats, bookingRef }: { showtime: Showtime; seats: Seat[]; bookingRef?: string }) {
  return (
    <article className="ticket" aria-label="Ticket">
      <div className="ticket-main">
        <p className="house">Lantern Picture House</p>
        <h2>{showtime.movie.title}</h2>
        <dl>
          <dt>When</dt>
          <dd>
            {formatDay(showtime.start)}, {formatTime(showtime.start)}
          </dd>
          <dt>Where</dt>
          <dd>{showtime.room}</dd>
          <dt>Seats</dt>
          <dd>{seats.map(seatLabel).join(', ')}</dd>
          <dt>Total</dt>
          <dd>{total(showtime, seats.length)}</dd>
        </dl>
      </div>
      <div className="stub">
        <span>Admit</span>
        <strong>{seats.length}</strong>
        {bookingRef && (
          <>
            <span>Ref</span>
            <b className="ref">{bookingRef}</b>
          </>
        )}
      </div>
    </article>
  )
}
