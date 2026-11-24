import { Link, useSearchParams } from "react-router";
import {
  dayKey,
  findSessions,
  formatDay,
  formatRuntime,
  formatTime,
  fromParams,
  slug,
  type Movie,
  type Showtime,
} from "@/lib/data";
import { MoviesStatus } from "@/components/MoviesStatus";
import { Poster, TmdbCredit } from "@/components/Poster";
import { QuickBook } from "@/components/QuickBook";
import { useTitle } from "@/lib/routes";

// What the quick-book bar found: each film with its sessions, by cinema then day.
export default function Sessions({
  movies,
  onRetry,
}: {
  movies: Movie[] | null | undefined;
  onRetry: () => void;
}) {
  const [params] = useSearchParams();
  const q = fromParams(params);
  useTitle("Sessions");
  const now = new Date();
  const found = movies ? findSessions(movies, q, now) : [];
  const films = movies?.filter((m) => found.some((s) => s.movie === m)) ?? [];
  const today = dayKey(now);

  return (
    <>
      {/* Remounted per search, so it always shows the one in the URL. */}
      <QuickBook key={params.toString()} movies={movies} initial={q} />
      <main>
        <h1 tabIndex={-1}>Sessions</h1>
        <MoviesStatus movies={movies} onRetry={onRetry} />
        {movies && !films.length && (
          <p className="none">
            No sessions match. Try another cinema or day.{" "}
            <Link className="link" to="/movies">
              See everything that's on
            </Link>
          </p>
        )}
        <ul className="listing">
          {films.map((movie) => {
            // Grouped by cinema then day; found is already soonest first.
            const groups = new Map<string, Showtime[]>();
            for (const s of found
              .filter((s) => s.movie === movie)
              .sort((a, b) => a.cinema.localeCompare(b.cinema))) {
              const day = dayKey(s.start) === today ? "Today" : formatDay(s.start);
              const key = `${s.cinema} · ${day}`;
              groups.set(key, [...(groups.get(key) ?? []), s]);
            }
            return (
              <li key={movie.movieID} className="film">
                <Link to={`/movies/${slug(movie.title)}`} tabIndex={-1} aria-hidden="true">
                  <Poster movie={movie} />
                </Link>
                <div>
                  <h2>
                    <Link to={`/movies/${slug(movie.title)}`}>{movie.title}</Link>
                  </h2>
                  <p className="meta">
                    <span>{movie.genre}</span>
                    <span>{formatRuntime(movie.runtimeMinutes)}</span>
                    <span className="rating">{movie.rating}</span>
                  </p>
                  <p className="synopsis">{movie.synopsis}</p>
                  {[...groups].map(([where, sessions]) => (
                    <section key={where} className="where">
                      <h3>{where}</h3>
                      <div className="times">
                        {sessions.map((s) => (
                          <Link
                            key={s.id}
                            to={`/sessions/${s.id}`}
                            state={{ pushed: true }}
                            aria-label={`${movie.title} at ${s.cinema}, ${formatTime(s.start)}, ${s.format}`}
                          >
                            {formatTime(s.start)}
                            {s.format !== "Standard" && <small> {s.format}</small>}
                          </Link>
                        ))}
                      </div>
                    </section>
                  ))}
                </div>
              </li>
            );
          })}
        </ul>
      </main>
      {films.some((m) => m.posterPath) && <TmdbCredit />}
    </>
  );
}
