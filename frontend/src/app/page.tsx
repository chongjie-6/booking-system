import {
  formatDay,
  formatRuntime,
  formatTime,
  slug,
  showtimesFor,
  upcomingDays,
  type Movie,
  type Showtime,
} from "@/lib/data";
import { Link, useParams } from "react-router";
import { MoviesStatus } from "@/components/MoviesStatus";
import { Poster, TmdbCredit } from "@/components/Poster";
import { Hero } from "@/components/Hero";
import { useTitle } from "@/lib/routes";

export default function Films({
  movies,
  day,
  onRetry,
  onDay,
  onPick,
}: {
  movies: Movie[] | null | undefined;
  day: number;
  onRetry: () => void;
  onDay: (i: number) => void;
  onPick: (s: Showtime) => void;
}) {
  const { movie: movieSlug } = useParams();
  const days = upcomingDays();
  const now = new Date();
  const shown = movieSlug
    ? movies?.filter((m) => slug(m.title) === movieSlug)
    : movies;
  useTitle((movieSlug && shown?.[0]?.title) || "Now showing");

  return (
    <>
      {!movieSlug && <Hero movies={movies?.filter((m) => m.featured) ?? []} />}
      <main>
        {movieSlug && (
          <Link className="back" to="/">
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
