import {
  formatDay,
  formatRuntime,
  formatTime,
  slug,
  showtimesFor,
  upcomingDays,
  type Movie,
} from "@/lib/data";
import { Link, useParams, useSearchParams } from "react-router";
import { MoviesStatus } from "@/components/MoviesStatus";
import { Poster, TmdbCredit } from "@/components/Poster";
import { useTitle } from "@/lib/routes";

export default function Films({
  movies,
  onRetry,
}: {
  movies: Movie[] | null | undefined;
  onRetry: () => void;
}) {
  const { movie: movieSlug } = useParams();
  const days = upcomingDays();
  // The day lives in the URL (?day=1), so refresh, Back and shared links keep it.
  const [params, setParams] = useSearchParams();
  const day = Math.max(0, Math.min(Math.trunc(Number(params.get("day"))) || 0, days.length - 1));
  const onDay = (i: number) => setParams(i ? { day: String(i) } : {}, { replace: true });
  const now = new Date();
  const shown = movieSlug
    ? movies?.filter((m) => slug(m.title) === movieSlug)
    : movies;
  useTitle((movieSlug && shown?.[0]?.title) || "Now showing");

  return (
    <>
      <main>
        {movieSlug && (
          <Link className="back" to="/movies">
            All films
          </Link>
        )}
        <h1 tabIndex={-1}>
          {movieSlug ? (shown?.[0]?.title ?? "Film") : "Now showing"}
        </h1>
        {movieSlug && (
          <nav className="days" aria-label="Day">
            {days.map((d, i) => (
              <button key={i} aria-pressed={i === day} onClick={() => onDay(i)}>
                {i === 0 ? "Today" : i === 1 ? "Tomorrow" : formatDay(d)}
              </button>
            ))}
          </nav>
        )}

        <MoviesStatus movies={movies} onRetry={onRetry} />
        {movieSlug && shown?.length === 0 && (
          <p className="none">
            That film isn't showing any more.{" "}
            <Link className="link" to="/">
              See what's showing
            </Link>
          </p>
        )}
        {!movieSlug && movies?.length === 0 && (
          <p className="none">Nothing is showing in the next few days.</p>
        )}

        <ul className="listing">
          {shown?.map((movie) => {
            const sessions = showtimesFor(movie, days[day]).filter(
              (s) => s.start > now,
            );
            return (
              <li key={movie.movieID} className="film">
                {movieSlug ? (
                  <Poster movie={movie} />
                ) : (
                  <Link
                    to={`/movies/${slug(movie.title)}`}
                    tabIndex={-1}
                    aria-hidden="true"
                  >
                    <Poster movie={movie} />
                  </Link>
                )}
                <div>
                  {!movieSlug && (
                    <h2>
                      <Link to={`/movies/${slug(movie.title)}`}>
                        {movie.title}
                      </Link>
                    </h2>
                  )}
                  <p className="meta">
                    <span>{movie.genre}</span>
                    <span>{formatRuntime(movie.runtimeMinutes)}</span>
                    <span className="rating">{movie.rating}</span>
                  </p>
                  <p className="synopsis">{movie.synopsis}</p>
                  {!movieSlug ? null : sessions.length ? (
                    <div className="times">
                      {sessions.map((s) => (
                        <Link
                          key={s.id}
                          to={`/sessions/${s.id}`}
                          state={{ pushed: true }}
                          aria-label={`${movie.title} at ${formatTime(s.start)}`}
                        >
                          {formatTime(s.start)}
                        </Link>
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
                          <button
                            className="link"
                            onClick={() => onDay(day + 1)}
                          >
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
      {shown?.some((m) => m.posterPath) && <TmdbCredit />}
    </>
  );
}
