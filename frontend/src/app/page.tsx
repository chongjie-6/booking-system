import type { CSSProperties } from "react";
import {
  formatDay,
  formatTime,
  hue,
  posterURL,
  slug,
  showtimesFor,
  upcomingDays,
  type Movie,
  type Showtime,
} from "@/lib/data";
import { Link, useParams } from "react-router";
import { MoviesStatus } from "@/components/MoviesStatus";
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
    <main>
      {movieSlug && (
        <Link className="back" to="/">
          All films
        </Link>
      )}
      <h1 tabIndex={-1}>
        {movieSlug ? (shown?.[0]?.title ?? "Film") : "Now showing"}
      </h1>
      <nav className="days" aria-label="Day">
        {days.map((d, i) => (
          <button key={i} aria-pressed={i === day} onClick={() => onDay(i)}>
            {i === 0 ? "Today" : i === 1 ? "Tomorrow" : formatDay(d)}
          </button>
        ))}
      </nav>

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
              <div
                className="poster"
                style={{ "--hue": hue(movie.movieID) } as CSSProperties}
                aria-hidden="true"
              >
                {movie.posterPath ? (
                  <img src={posterURL(movie.posterPath)} alt="" loading="lazy" />
                ) : (
                  <span>{movie.title}</span>
                )}
              </div>
              <div>
                {!movieSlug && (
                  <h2>
                    <Link to={`/movies/${slug(movie.title)}`}>{movie.title}</Link>
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
