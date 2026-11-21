import {
  useEffect,
  useRef,
  useState,
  type CSSProperties,
  type Dispatch,
  type MouseEvent as ReactMouseEvent,
  type ReactNode,
  type SetStateAction,
} from "react";
import { api, ApiError, type User } from "./api";
import { Auth } from "./Auth";
import {
  MAX_SEATS,
  formatDay,
  formatTime,
  findShowtime,
  hue,
  seatLabel,
  showtimesFor,
  total,
  upcomingDays,
  type Booking,
  type Movie,
  type Seat,
  type Showtime,
} from "./data";

// Every page has its own URL so it can be bookmarked or shared.
type Route =
  | { step: "films"; movieID?: string }
  | { step: "seats" | "checkout"; eventID: string }
  | { step: "done"; ref: string };

function parse(path: string): Route {
  const [section, id, sub] = path.split("/").filter(Boolean);
  if (section === "movies" && id) return { step: "films", movieID: id };
  if (section === "sessions" && id)
    return { step: sub === "checkout" ? "checkout" : "seats", eventID: id };
  if (section === "bookings" && id) return { step: "done", ref: id };
  return { step: "films" };
}

// What each history entry keeps, so Back, Forward and refresh return with the same seats.
// pushed marks entries the app added, which a back button can safely return from.
type Entry = { seats?: Seat[]; booking?: Booking; pushed?: boolean };

type Link = (to: string) => {
  href: string;
  onClick: (e: ReactMouseEvent) => void;
};

const titles: Record<Route["step"], string> = {
  films: "Now showing",
  seats: "Choose seats",
  checkout: "Checkout",
  done: "Booking confirmed",
};

// The hold timer warns this long before the seats are released. WCAG 2.2.1 asks for at least 20 seconds.
const HOLD_WARNING_SECONDS = 120;

// A reset email links to /#reset=<token>. Read it once, then drop it from the address bar and history.
const resetToken =
  new URLSearchParams(location.hash.slice(1)).get("reset") ?? undefined;
if (resetToken)
  history.replaceState(null, "", location.pathname + location.search);
const restored = history.state as Entry | null;

// Set when the user leaves checkout. The seat map waits for it, so the user's own seats don't come back as taken.
let releasing: Promise<unknown> = Promise.resolve();

export default function App() {
  const [path, setPath] = useState(location.pathname);
  const [signingIn, setSigningIn] = useState(!!resetToken);
  const [day, setDay] = useState(0);
  const [seats, setSeats] = useState<Seat[]>(() => restored?.seats ?? []);
  const [booking, setBooking] = useState(restored?.booking);
  // undefined while loading, null if it failed
  const [movies, setMovies] = useState<Movie[] | null>();
  const [moviesAttempt, setMoviesAttempt] = useState(0);
  // undefined while the first lookup runs, null when signed out
  const [user, setUser] = useState<User | null>();
  const [signOutFailed, setSignOutFailed] = useState(false);
  const navigated = useRef(false);

  let route = parse(path);
  // Checkout opened from a bookmark has no seats chosen yet.
  if (route.step === "checkout" && !seats.length)
    route = { step: "seats", eventID: route.eventID };
  const showtime =
    "eventID" in route && movies
      ? findShowtime(movies, route.eventID)
      : undefined;
  const movie =
    route.step === "films"
      ? movies?.find((m) => m.movieID === route.movieID)
      : undefined;
  const title = movie?.title ?? titles[route.step];

  useEffect(() => {
    api<User>("/user").then(setUser, () => setUser(null));
  }, []);

  useEffect(() => {
    api<Movie[]>("/movies").then(setMovies, () => setMovies(null));
  }, [moviesAttempt]);

  useEffect(() => {
    history.replaceState({ ...history.state, seats }, "");
  }, [path, seats]);

  useEffect(() => {
    document.title = `${title} · Lantern Picture House`;
  }, [title]);

  // After a page change, move focus to the new heading so screen readers announce it. Not on first load.
  useEffect(() => {
    if (!navigated.current) return;
    window.scrollTo(0, 0);
    document
      .querySelector<HTMLElement>("main h1")
      ?.focus({ preventScroll: true });
  }, [path]);

  useEffect(() => {
    function onPop() {
      // Back from a confirmation goes to the films, not to a seat map where the user's seats are now booked.
      if (route.step === "done") {
        go("/", true);
        setSeats([]);
        return;
      }
      const entry = history.state as Entry | null;
      leave(parse(location.pathname));
      setPath(location.pathname);
      setSeats(entry?.seats ?? []);
      setBooking(entry?.booking);
    }
    addEventListener("popstate", onPop);
    return () => removeEventListener("popstate", onPop);
  });

  // Leaving checkout any way but by booking releases the holds, so other people can take the seats straight away.
  function leave(next: Route) {
    navigated.current = true;
    if (route.step !== "checkout" || next.step === "done" || !user) return;
    const body = { seatIDs: seats.map((s) => s.seatID) };
    releasing = api(`/events/${route.eventID}/hold`, {
      method: "DELETE",
      body,
    }).catch(() => {});
  }

  function go(to: string, replace = false) {
    leave(parse(to));
    if (replace) history.replaceState(history.state, "", to);
    else history.pushState({ pushed: true }, "", to);
    setPath(to);
  }

  function back(fallback: string) {
    if ((history.state as Entry | null)?.pushed) history.back();
    else go(fallback);
  }

  // Real links, so a page can be opened in a new tab; a plain click stays in the app.
  const link: Link = (to) => ({
    href: to,
    onClick(e) {
      if (e.button || e.metaKey || e.ctrlKey || e.shiftKey || e.altKey) return;
      e.preventDefault();
      go(to);
    },
  });

  function retryMovies() {
    setMovies(undefined);
    setMoviesAttempt((a) => a + 1);
  }

  function signOut() {
    setSignOutFailed(false);
    api("/auth/logout", { method: "POST" }).then(
      () => setUser(null),
      () => setSignOutFailed(true),
    );
  }

  function booked(showtime: Showtime, user: User, ref: string) {
    const b = { ref, name: user.firstName, email: user.email, showtime, seats };
    history.replaceState({ ...history.state, booking: b }, "");
    go(`/bookings/${ref}`, true);
    setBooking(b);
    setSeats([]);
  }

  return (
    <>
      <header className="masthead">
        <a className="wordmark" {...link("/")}>
          Lantern
        </a>
        {user ? (
          <div className="account">
            {signOutFailed && (
              <span className="error" role="alert">
                Couldn't sign you out. Try again.
              </span>
            )}
            <span>{user.firstName}</span>
            <button className="link" onClick={signOut}>
              Sign out
            </button>
          </div>
        ) : (
          user === null && (
            <button className="link signin" onClick={() => setSigningIn(true)}>
              <svg width="20" height="20" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" aria-hidden="true">
                <circle cx="12" cy="8" r="4" />
                <path d="M4 21a8 8 0 0 1 16 0" />
              </svg>
              Sign in
            </button>
          )
        )}
      </header>

      {user === null && (
        <Drawer open={signingIn} onClose={() => setSigningIn(false)}>
          <Auth
            mode={resetToken ? "reset" : "login"}
            resetToken={resetToken}
            onSignedIn={(u) => {
              setUser(u);
              setSigningIn(false);
            }}
          />
        </Drawer>
      )}

      {route.step === "films" && (
        <Films
          movies={movies}
          movieID={route.movieID}
          day={day}
          link={link}
          onRetry={retryMovies}
          onDay={setDay}
          onPick={(showtime) => {
            setSeats([]);
            go(`/sessions/${showtime.id}`);
          }}
        />
      )}
      {"eventID" in route && !showtime && (
        <main>
          <h1 tabIndex={-1}>{titles[route.step]}</h1>
          <MoviesStatus movies={movies} onRetry={retryMovies} />
          {movies && (
            <p className="none">
              This session isn't on any more.{" "}
              <a className="link" {...link("/")}>
                See what's showing
              </a>
            </p>
          )}
        </main>
      )}
      {route.step === "seats" && showtime && (
        <SeatPicker
          showtime={showtime}
          seats={seats}
          onSeats={setSeats}
          onBack={() => back("/")}
          onContinue={() => go(`/sessions/${showtime.id}/checkout`)}
        />
      )}
      {route.step === "checkout" && showtime && (
        <Checkout
          showtime={showtime}
          seats={seats}
          user={user}
          onSignedIn={setUser}
          onBack={() => back(`/sessions/${showtime.id}`)}
          onFilms={() => go("/")}
          onBooked={(u, ref) => booked(showtime, u, ref)}
        />
      )}
      {route.step === "done" && (
        <Done
          bookingRef={route.ref}
          booking={booking?.ref === route.ref ? booking : undefined}
          onAgain={() => go("/")}
        />
      )}
    </>
  );
}

function MoviesStatus({
  movies,
  onRetry,
}: {
  movies: Movie[] | null | undefined;
  onRetry: () => void;
}) {
  if (movies === undefined)
    return (
      <p className="none" role="status">
        Loading films…
      </p>
    );
  if (movies === null)
    return (
      <p className="error" role="alert">
        Couldn't load the films.{" "}
        <button className="link" onClick={onRetry}>
          Try again
        </button>
      </p>
    );
  return null;
}

// A modal panel that slides in from the right. Closes on Esc, the close button, or a click on the backdrop.
function Drawer({
  open,
  onClose,
  children,
}: {
  open: boolean;
  onClose: () => void;
  children: ReactNode;
}) {
  const ref = useRef<HTMLDialogElement>(null);

  useEffect(() => {
    if (open) ref.current?.showModal();
    else ref.current?.close();
  }, [open]);

  return (
    <dialog
      ref={ref}
      className="drawer"
      onClose={onClose}
      onClick={(e) => e.target === e.currentTarget && ref.current?.close()}
    >
      <div className="drawer-body">
        <button
          className="close"
          aria-label="Close"
          onClick={() => ref.current?.close()}
        >
          <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" aria-hidden="true">
            <path d="M6 6l12 12M18 6L6 18" />
          </svg>
        </button>
        {children}
      </div>
    </dialog>
  );
}

function Films({
  movies,
  movieID,
  day,
  link,
  onRetry,
  onDay,
  onPick,
}: {
  movies: Movie[] | null | undefined;
  movieID?: string;
  day: number;
  link: Link;
  onRetry: () => void;
  onDay: (i: number) => void;
  onPick: (s: Showtime) => void;
}) {
  const days = upcomingDays();
  const now = new Date();
  const shown = movieID
    ? movies?.filter((m) => m.movieID === movieID)
    : movies;

  return (
    <main>
      {movieID && (
        <a className="back" {...link("/")}>
          All films
        </a>
      )}
      <h1 tabIndex={-1}>
        {movieID ? (shown?.[0]?.title ?? "Film") : "Now showing"}
      </h1>
      <nav className="days" aria-label="Day">
        {days.map((d, i) => (
          <button key={i} aria-pressed={i === day} onClick={() => onDay(i)}>
            {i === 0 ? "Today" : i === 1 ? "Tomorrow" : formatDay(d)}
          </button>
        ))}
      </nav>

      <MoviesStatus movies={movies} onRetry={onRetry} />
      {movieID && shown?.length === 0 && (
        <p className="none">
          That film isn't showing any more.{" "}
          <a className="link" {...link("/")}>
            See what's showing
          </a>
        </p>
      )}
      {!movieID && movies?.length === 0 && (
        <p className="none">Nothing is showing in the next few days.</p>
      )}

      <ul className="listing">
        {shown?.map((movie) => {
          const sessions = showtimesFor(movie, days[day]).filter(
            (s) => s.start > now,
          );
          return (
            <li key={movie.movieID} className="film">
              <div
                className="poster"
                style={{ "--hue": hue(movie.movieID) } as CSSProperties}
                aria-hidden="true"
              >
                <span>{movie.title}</span>
              </div>
              <div>
                {!movieID && (
                  <h2>
                    <a {...link(`/movies/${movie.movieID}`)}>{movie.title}</a>
                  </h2>
                )}
                <p className="meta">
                  <span>{movie.genre}</span>
                  <span>
                    {Math.floor(movie.runtimeMinutes / 60)}h{" "}
                    {movie.runtimeMinutes % 60}m
                  </span>
                  <span className="rating">{movie.rating}</span>
                </p>
                <p className="synopsis">{movie.synopsis}</p>
                {sessions.length ? (
                  <div className="times">
                    {sessions.map((s) => (
                      <button
                        key={s.id}
                        aria-label={`${movie.title} at ${formatTime(s.start)}`}
                        onClick={() => onPick(s)}
                      >
                        {formatTime(s.start)}
                      </button>
                    ))}
                  </div>
                ) : (
                  <p className="none">
                    {day === 0
                      ? "No more sessions today."
                      : `No sessions on ${formatDay(days[day])}.`}
                    {day < days.length - 1 && (
                      <>
                        {" "}
                        <button className="link" onClick={() => onDay(day + 1)}>
                          See{" "}
                          {day === 0 ? "tomorrow" : formatDay(days[day + 1])}
                        </button>
                      </>
                    )}
                  </p>
                )}
              </div>
            </li>
          );
        })}
      </ul>
    </main>
  );
}

function SeatPicker({
  showtime,
  seats,
  onSeats,
  onBack,
  onContinue,
}: {
  showtime: Showtime;
  seats: Seat[];
  onSeats: Dispatch<SetStateAction<Seat[]>>;
  onBack: () => void;
  onContinue: () => void;
}) {
  // undefined while loading, null if it failed
  const [all, setAll] = useState<Seat[] | null>();
  const [attempt, setAttempt] = useState(0);
  // The seats the user arrived with, e.g. coming back from checkout, and which of those someone else has taken since.
  const [arrivedWith] = useState(seats);
  const [lost, setLost] = useState<Seat[]>([]);
  const atMax = seats.length >= MAX_SEATS;
  const types = new Set(all?.map((s) => s.type));
  const soldOut = !!all?.length && all.every((s) => s.taken);

  useEffect(() => {
    releasing
      .then(() => api<Seat[]>(`/events/${showtime.id}/seats`))
      .then(
        (all) => {
          setAll(all);
          const taken = new Set(
            all.filter((s) => s.taken).map((s) => s.seatID),
          );
          setLost(arrivedWith.filter((s) => taken.has(s.seatID)));
          onSeats((seats) => seats.filter((s) => !taken.has(s.seatID)));
        },
        () => setAll(null),
      );
  }, [showtime.id, onSeats, arrivedWith, attempt]);

  // The API sends seats ordered by row then number.
  const rows: Seat[][] = [];
  for (const seat of all ?? []) {
    if (rows.at(-1)?.[0].row === seat.row) rows.at(-1)!.push(seat);
    else rows.push([seat]);
  }

  function toggle(seat: Seat) {
    onSeats((seats) => {
      if (seats.some((s) => s.seatID === seat.seatID))
        return seats.filter((s) => s.seatID !== seat.seatID);
      if (seats.length >= MAX_SEATS) return seats;
      return [...seats, seat].sort(
        (a, b) => a.row.localeCompare(b.row) || a.number - b.number,
      );
    });
  }

  return (
    <main>
      <button className="back" onClick={onBack}>
        Back to films
      </button>
      <h1 tabIndex={-1}>{showtime.movie.title}</h1>
      <p className="meta">
        <span>
          {formatDay(showtime.start)}, {formatTime(showtime.start)}
        </span>
        <span>{showtime.room}</span>
      </p>

      {all === undefined && (
        <p className="none" role="status">
          Loading seats…
        </p>
      )}
      {all === null && (
        <p className="error" role="alert">
          Couldn't load the seats.{" "}
          <button
            className="link"
            onClick={() => {
              setAll(undefined);
              setAttempt((a) => a + 1);
            }}
          >
            Try again
          </button>
        </p>
      )}
      {lost.length > 0 && (
        <p className="hint" role="status">
          Someone else just took {lost.map(seatLabel).join(", ")}, so{" "}
          {lost.length === 1 ? "it's" : "they're"} no longer in your selection.
        </p>
      )}
      {soldOut && (
        <p className="none">
          This session is sold out.{" "}
          <button className="link" onClick={onBack}>
            Choose another session
          </button>
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
                const isTaken = seat.taken;
                const selected = seats.some((s) => s.seatID === seat.seatID);
                return (
                  <button
                    key={seat.seatID}
                    className="seat"
                    data-type={seat.type}
                    disabled={isTaken}
                    aria-pressed={selected}
                    aria-label={`Row ${seat.row}, seat ${seat.number}${seat.type === "standard" ? "" : `, ${seat.type}`}${isTaken ? ", taken" : ""}`}
                    onClick={() => toggle(seat)}
                  >
                    {selected
                      ? seat.number
                      : seat.type === "wheelchair" && "♿︎"}
                  </button>
                );
              })}
              <span className="row-label" aria-hidden="true">
                {row[0].row}
              </span>
            </div>
          ))}
        </div>
      </div>

      {all && (
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
          {types.has("wheelchair") && (
            <li>
              <span className="swatch" data-type="wheelchair">
                ♿&#xFE0E;
              </span>{" "}
              Wheelchair space
            </li>
          )}
          {types.has("companion") && (
            <li>
              <span className="swatch" data-type="companion" /> Companion seat
            </li>
          )}
          {types.has("premium") && (
            <li>
              <span className="swatch" data-type="premium" /> Premium
            </li>
          )}
        </ul>
      )}

      <footer className="tray">
        <div aria-live="polite">
          <p>
            {seats.length
              ? `${seats.length} ${seats.length === 1 ? "seat" : "seats"}: ${seats.map(seatLabel).join(", ")}`
              : `Choose up to ${MAX_SEATS} seats.`}
          </p>
          {atMax && (
            <p className="hint">That's the most seats one booking can hold.</p>
          )}
        </div>
        <strong className="total">{total(showtime, seats.length)}</strong>
        <button
          className="primary"
          disabled={!seats.length}
          onClick={onContinue}
        >
          Continue to checkout
        </button>
      </footer>
    </main>
  );
}

function Checkout({
  showtime,
  seats,
  user,
  onSignedIn,
  onBack,
  onFilms,
  onBooked,
}: {
  showtime: Showtime;
  seats: Seat[];
  user: User | null | undefined;
  onSignedIn: (u: User) => void;
  onBack: () => void;
  onFilms: () => void;
  onBooked: (u: User, ref: string) => void;
}) {
  const [error, setError] = useState<Error | null>(null);
  const [busy, setBusy] = useState(false);
  const [heldUntil, setHeldUntil] = useState<number>();
  // Bumped to hold the seats again, e.g. when the user asks for more time.
  const [holds, setHolds] = useState(0);
  const path = `/events/${showtime.id}`;
  const status = error instanceof ApiError ? error.status : 0;

  // Hold the seats while the user checks out, once they're signed in.
  useEffect(() => {
    if (!user) return;
    const seatIDs = seats.map((s) => s.seatID);
    api<{ holdSeconds: number }>(`/events/${showtime.id}/hold`, {
      method: "PUT",
      body: { seatIDs },
    }).then((r) => {
      setHeldUntil(Date.now() + r.holdSeconds * 1000);
      setError(null);
    }, setError);
  }, [user, showtime.id, seats, holds]);

  async function confirm(u: User) {
    setBusy(true);
    setError(null);
    try {
      const body = { seatIDs: seats.map((s) => s.seatID) };
      const { ref } = await api<{ ref: string }>(`${path}/bookings`, {
        method: "POST",
        body,
      });
      onBooked(u, ref);
    } catch (err) {
      setError(err as Error);
      setBusy(false);
    }
  }

  return (
    <main>
      <button className="back" onClick={onBack} disabled={busy}>
        Change seats
      </button>
      <h1 tabIndex={-1}>Confirm your booking</h1>
      <div className="checkout">
        {user ? (
          <div className="fields">
            <p>
              Booking as {user.firstName} {user.lastName}. We'll send your
              tickets to {user.email}.
            </p>
            {heldUntil && !error && (
              <HoldTimer
                until={heldUntil}
                onExtend={() => setHolds((h) => h + 1)}
              />
            )}
            {error && (
              <p className="error" role="alert">
                {error.message}
              </p>
            )}
            {status === 409 ? (
              <button className="primary" onClick={onBack}>
                Choose other seats
              </button>
            ) : status === 404 ? (
              <button className="primary" onClick={onFilms}>
                Choose another session
              </button>
            ) : (
              <button
                className="primary"
                onClick={() => confirm(user)}
                disabled={busy}
              >
                {busy ? "Booking…" : error ? "Try again" : "Confirm booking"}
              </button>
            )}
          </div>
        ) : user === null ? (
          <Auth onSignedIn={onSignedIn} />
        ) : (
          <p className="none" role="status">
            Loading…
          </p>
        )}
        <Ticket showtime={showtime} seats={seats} />
      </div>
    </main>
  );
}

// Counts down the seat hold. Only the warning is a live region, so screen readers hear it once, not every second.
function HoldTimer({
  until,
  onExtend,
}: {
  until: number;
  onExtend: () => void;
}) {
  const [now, setNow] = useState(Date.now);

  // Tick just after each second boundary of `until`, so the clock never skips or repeats a second.
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout>;
    const schedule = () => {
      const ms = (((until - Date.now()) % 1000) + 1000) % 1000;
      timer = setTimeout(() => {
        const t = Date.now();
        setNow(t);
        if (t < until) schedule();
      }, ms + 20);
    };
    schedule();
    return () => clearTimeout(timer);
  }, [until]);

  const left = Math.max(0, Math.ceil((until - now) / 1000));
  const clock = `${Math.floor(left / 60)}:${String(left % 60).padStart(2, "0")}`;

  return (
    <div className={left <= HOLD_WARNING_SECONDS ? "hold warn" : "hold"}>
      <p>
        {left ? (
          <>
            Seats held for <strong>{clock}</strong>
          </>
        ) : (
          "Your seats are no longer held."
        )}
      </p>
      {left <= HOLD_WARNING_SECONDS && (
        <>
          <p role="alert">
            {left
              ? "Your seats will be released soon."
              : "Someone else could book them now."}
          </p>
          <button className="link" onClick={onExtend}>
            {left ? "I need more time" : "Hold my seats again"}
          </button>
        </>
      )}
    </div>
  );
}

// booking is missing when the page is opened from a bookmark, since only this tab's history holds it.
function Done({
  bookingRef,
  booking,
  onAgain,
}: {
  bookingRef: string;
  booking?: Booking;
  onAgain: () => void;
}) {
  return (
    <main className="done">
      <h1 tabIndex={-1}>Booking confirmed</h1>
      {booking ? (
        <>
          <p>
            {booking.name}, your tickets are on their way to {booking.email}.
            Show reference {booking.ref} at the door.
          </p>
          <Ticket
            showtime={booking.showtime}
            seats={booking.seats}
            bookingRef={booking.ref}
          />
        </>
      ) : (
        <p>
          Your tickets were emailed to you. Show reference {bookingRef} at the
          door.
        </p>
      )}
      <button className="primary" onClick={onAgain}>
        Book another film
      </button>
    </main>
  );
}

function Ticket({
  showtime,
  seats,
  bookingRef,
}: {
  showtime: Showtime;
  seats: Seat[];
  bookingRef?: string;
}) {
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
          <dd>{seats.map(seatLabel).join(", ")}</dd>
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
  );
}
