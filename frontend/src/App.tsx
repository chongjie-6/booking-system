import { Fragment, useState, type CSSProperties, type Dispatch, type SetStateAction } from 'react'
import {
  AISLES_AFTER,
  MAX_SEATS,
  MOVIES,
  PRICE,
  ROWS,
  SEATS_PER_ROW,
  formatDay,
  formatTime,
  money,
  showtimesFor,
  takenSeats,
  upcomingDays,
  type Booking,
  type Showtime,
} from './data'

type View =
  | { step: 'films' }
  | { step: 'seats'; showtime: Showtime }
  | { step: 'checkout'; showtime: Showtime }
  | { step: 'done'; booking: Booking }

export default function App() {
  const [view, setView] = useState<View>({ step: 'films' })
  const [day, setDay] = useState(0)
  const [seats, setSeats] = useState<string[]>([])
  const [booked, setBooked] = useState<Record<string, string[]>>({})

  function go(next: View) {
    setView(next)
    window.scrollTo(0, 0)
  }

  function confirm(showtime: Showtime, name: string, email: string) {
    const booking = { ref: crypto.randomUUID().slice(0, 6).toUpperCase(), name, email, showtime, seats }
    setBooked({ ...booked, [showtime.id]: [...(booked[showtime.id] ?? []), ...seats] })
    setSeats([])
    go({ step: 'done', booking })
  }

  return (
    <>
      <header className="masthead">
        <button className="wordmark" onClick={() => go({ step: 'films' })}>
          Lantern
        </button>
      </header>

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
          onBack={() => go({ step: 'seats', showtime: view.showtime })}
          onConfirm={(name, email) => confirm(view.showtime, name, email)}
        />
      )}
      {view.step === 'done' && <Done booking={view.booking} onAgain={() => go({ step: 'films' })} />}
    </>
  )
}

function Films({ day, onDay, onPick }: { day: number; onDay: (i: number) => void; onPick: (s: Showtime) => void }) {
  const days = upcomingDays()
  const now = new Date()

  return (
    <main>
      <h1>Now showing</h1>
      <nav className="days" aria-label="Day">
        {days.map((d, i) => (
          <button key={i} aria-pressed={i === day} onClick={() => onDay(i)}>
            {i === 0 ? 'Today' : i === 1 ? 'Tomorrow' : formatDay(d)}
          </button>
        ))}
      </nav>

      <ul className="listing">
        {MOVIES.map((movie) => {
          const sessions = showtimesFor(movie, days[day]).filter((s) => s.start > now)
          return (
            <li key={movie.id} className="film">
              <div className="poster" style={{ '--hue': movie.hue } as CSSProperties} aria-hidden="true">
                <span>{movie.title}</span>
              </div>
              <div>
                <h2>{movie.title}</h2>
                <p className="meta">
                  <span>{movie.genre}</span>
                  <span>
                    {Math.floor(movie.runtime / 60)}h {movie.runtime % 60}m
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
  seats: string[]
  onSeats: Dispatch<SetStateAction<string[]>>
  onBack: () => void
  onContinue: () => void
}) {
  const taken = new Set([...takenSeats(showtime.id), ...booked])
  const atMax = seats.length >= MAX_SEATS

  function toggle(seat: string) {
    onSeats((seats) => {
      if (seats.includes(seat)) return seats.filter((s) => s !== seat)
      if (seats.length >= MAX_SEATS) return seats
      return [...seats, seat].sort((a, b) => a.localeCompare(b, undefined, { numeric: true }))
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
        <span>Screen {showtime.movie.screen}</span>
      </p>

      <div className="auditorium">
        <div className="screen" aria-hidden="true" />
        <div className="seats" role="group" aria-label="Seats">
          {ROWS.map((row) => (
            <div className="row" key={row}>
              <span className="row-label" aria-hidden="true">
                {row}
              </span>
              {Array.from({ length: SEATS_PER_ROW }, (_, i) => {
                const n = i + 1
                const id = `${row}${n}`
                const isTaken = taken.has(id)
                const selected = seats.includes(id)
                return (
                  <Fragment key={id}>
                    <button
                      className="seat"
                      disabled={isTaken}
                      aria-pressed={selected}
                      aria-label={`Row ${row}, seat ${n}${isTaken ? ', taken' : ''}`}
                      onClick={() => toggle(id)}
                    >
                      {selected && n}
                    </button>
                    {AISLES_AFTER.includes(n) && <span className="aisle" />}
                  </Fragment>
                )
              })}
              <span className="row-label" aria-hidden="true">
                {row}
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
              ? `${seats.length} ${seats.length === 1 ? 'seat' : 'seats'}: ${seats.join(', ')}`
              : `Choose up to ${MAX_SEATS} seats.`}
          </p>
          {atMax && <p className="hint">That's the most seats one booking can hold.</p>}
        </div>
        <strong className="total">{money.format(seats.length * PRICE)}</strong>
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
  onBack,
  onConfirm,
}: {
  showtime: Showtime
  seats: string[]
  onBack: () => void
  onConfirm: (name: string, email: string) => void
}) {
  return (
    <main>
      <button className="back" onClick={onBack}>
        Change seats
      </button>
      <h1>Confirm your booking</h1>
      <div className="checkout">
        <form
          onSubmit={(e) => {
            e.preventDefault()
            const form = new FormData(e.currentTarget)
            onConfirm(String(form.get('name')).trim(), String(form.get('email')).trim())
          }}
        >
          <label>
            Name
            <input name="name" required autoComplete="name" />
          </label>
          <label>
            Email
            <input name="email" type="email" required autoComplete="email" />
          </label>
          <p className="fine">We'll send your tickets to this address.</p>
          <button className="primary">Confirm booking</button>
        </form>
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

function Ticket({ showtime, seats, bookingRef }: { showtime: Showtime; seats: string[]; bookingRef?: string }) {
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
          <dt>Screen</dt>
          <dd>{showtime.movie.screen}</dd>
          <dt>Seats</dt>
          <dd>{seats.join(', ')}</dd>
          <dt>Total</dt>
          <dd>{money.format(seats.length * PRICE)}</dd>
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
