import {
  useEffect,
  useLayoutEffect,
  useRef,
  useState,
  type ReactNode,
} from "react";
import {
  Link,
  Navigate,
  Route,
  Routes,
  matchPath,
  useLocation,
  useNavigate,
  useNavigationType,
  useParams,
} from "react-router";
import { api, releaseHold, type User } from "@/lib/api";
import { Auth } from "@/components/Auth";
import { findShowtime, type Movie, type Showtime } from "@/lib/data";
import { Drawer } from "@/components/Drawer";
import { MoviesStatus } from "@/components/MoviesStatus";
import Checkout from "@/app/sessions/[eventID]/checkout/page";
import Done from "@/app/bookings/[ref]/page";
import Films from "@/app/page";
import SeatPicker from "@/app/sessions/[eventID]/page";
import { useTitle, type Entry } from "@/lib/routes";

// A reset email links to /#reset=<token>. Read it once, then drop it from the address bar and history.
const resetToken =
  new URLSearchParams(location.hash.slice(1)).get("reset") ?? undefined;
if (resetToken)
  history.replaceState(null, "", location.pathname + location.search);

export default function App() {
  const location = useLocation();
  const navigate = useNavigate();
  const navigationType = useNavigationType();
  const entry = location.state as Entry | null;
  const [signingIn, setSigningIn] = useState(!!resetToken);
  const [drawerOpens, setDrawerOpens] = useState(0);
  const [day, setDay] = useState(0);
  // undefined while loading, null if it failed
  const [movies, setMovies] = useState<Movie[] | null>();
  const [moviesAttempt, setMoviesAttempt] = useState(0);
  // undefined while the first lookup runs, null when signed out
  const [user, setUser] = useState<User | null>();
  const [signOutFailed, setSignOutFailed] = useState(false);
  const [shown, setShown] = useState(location);
  const [leftDone, setLeftDone] = useState(false);
  const previous = useRef(location);

  // Back from a confirmation goes to the films, not to a seat map where the user's seats are now booked.
  if (shown !== location) {
    setShown(location);
    setLeftDone(
      navigationType === "POP" && shown.pathname.startsWith("/bookings/"),
    );
  }

  useEffect(() => {
    api.get<User>("/user").then(setUser, () => setUser(null));
  }, []);

  useEffect(() => {
    api.get<Movie[]>("/movies").then(setMovies, () => setMovies(null));
  }, [moviesAttempt]);

  // A layout effect, so a hold is released before the seat map's own effect asks which seats are taken.
  useLayoutEffect(() => {
    const from = previous.current;
    previous.current = location;
    if (from.pathname === location.pathname) return;

    // Leaving checkout any way but by booking releases the holds, so other people can take the seats straight away.
    const checkout = matchPath("/sessions/:eventID/checkout", from.pathname);
    const held = (from.state as Entry | null)?.seats;
    if (checkout && held && user && !location.pathname.startsWith("/bookings/"))
      releaseHold(
        checkout.params.eventID!,
        held.map((s) => s.seatID),
      );

    // Move focus to the new heading so screen readers announce it.
    window.scrollTo(0, 0);
    document
      .querySelector<HTMLElement>("main h1")
      ?.focus({ preventScroll: true });
  }, [location, user]);

  function back(fallback: string) {
    if (entry?.pushed) navigate(-1);
    else navigate(fallback);
  }

  function retryMovies() {
    setMovies(undefined);
    setMoviesAttempt((a) => a + 1);
  }

  function signOut() {
    setSignOutFailed(false);
    api.post("/auth/logout").then(
      () => setUser(null),
      () => setSignOutFailed(true),
    );
  }

  const chosen = entry?.seats ?? [];

  return (
    <>
      <header className="masthead">
        <Link className="wordmark" to="/">
          Lantern
        </Link>
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
            <button className="link signin" onClick={() => {
                setDrawerOpens((n) => n + 1);
                setSigningIn(true);
              }}
            >
              <svg
                width="20"
                height="20"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                strokeWidth="2"
                strokeLinecap="round"
                aria-hidden="true"
              >
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
          {/* Remounted on each open so it starts on sign in; not on close, so it doesn't change mid slide-out. */}
          <Auth
            key={drawerOpens}
            mode={resetToken ? "reset" : "login"}
            resetToken={resetToken}
            onSignedIn={(u) => {
              setUser(u);
              setSigningIn(false);
            }}
          />
        </Drawer>
      )}

      {leftDone ? (
        <Navigate to="/" replace />
      ) : (
        <Routes>
          {["/", "/movies/:movie"].map((path) => (
            <Route
              key={path}
              path={path}
              element={
                <Films
                  movies={movies}
                  day={day}
                  onRetry={retryMovies}
                  onDay={setDay}
                  onPick={(s) =>
                    navigate(`/sessions/${s.id}`, { state: { pushed: true } })
                  }
                />
              }
            />
          ))}
          <Route
            path="/sessions/:eventID"
            element={
              <Session
                movies={movies}
                onRetry={retryMovies}
                title="Choose seats"
              >
                {(showtime) => (
                  <SeatPicker
                    showtime={showtime}
                    onBack={() => back("/")}
                    onContinue={(seats) =>
                      navigate(`/sessions/${showtime.id}/checkout`, {
                        state: { pushed: true, seats },
                      })
                    }
                  />
                )}
              </Session>
            }
          />
          <Route
            path="/sessions/:eventID/checkout"
            element={
              // Checkout opened from a bookmark has no seats chosen yet.
              chosen.length ? (
                <Session movies={movies} onRetry={retryMovies} title="Checkout">
                  {(showtime) => (
                    <Checkout
                      showtime={showtime}
                      seats={chosen}
                      user={user}
                      onSignedIn={setUser}
                      onBack={() => back(`/sessions/${showtime.id}`)}
                      onFilms={() => navigate("/")}
                      onBooked={(u, ref) => {
                        const booking = {
                          ref,
                          name: u.firstName,
                          email: u.email,
                          showtime,
                          seats: chosen,
                        };
                        navigate(`/bookings/${ref}`, {
                          replace: true,
                          state: { booking },
                        });
                      }}
                    />
                  )}
                </Session>
              ) : (
                <Navigate to=".." relative="path" replace />
              )
            }
          />
          <Route
            path="/bookings/:ref"
            element={<Done onAgain={() => navigate("/")} />}
          />
          <Route path="*" element={<Navigate to="/" replace />} />
        </Routes>
      )}
      <footer className="credits">
        Film details and posters from{" "}
        <a className="link" href="https://www.themoviedb.org">TMDB</a>. This product
        uses the TMDB API but is not endorsed or certified by TMDB.
      </footer>
    </>
  );
}

// Finds the session named in the URL, or says why it can't.
function Session({
  movies,
  onRetry,
  title,
  children,
}: {
  movies: Movie[] | null | undefined;
  onRetry: () => void;
  title: string;
  children: (showtime: Showtime) => ReactNode;
}) {
  const { eventID = "" } = useParams();
  const showtime = movies && findShowtime(movies, eventID);
  useTitle(title);
  if (showtime) return children(showtime);
  return (
    <main>
      <h1 tabIndex={-1}>{title}</h1>
      <MoviesStatus movies={movies} onRetry={onRetry} />
      {movies && (
        <p className="none">
          This session isn't on any more.{" "}
          <Link className="link" to="/">
            See what's showing
          </Link>
        </p>
      )}
    </main>
  );
}
