import {
  formatDay,
  formatRuntime,
  formatTime,
  MAX_CINEMAS,
  savedCinemas,
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
  soon,
  onRetry,
}: {
  movies: Movie[] | null | undefined;
  soon: Movie[] | undefined;
  onRetry: () => void;
}) {
  const { movie: movieSlug } = useParams();
  const now = new Date();
  const shown = movieSlug
    ? movies && soon && [...movies, ...soon].filter((m) => slug(m.title) === movieSlug)
    : movies;
  // A coming-soon film's days are the ones it has showings on; anything else gets today and the next two.
  const opening = !!shown?.[0] && !movies?.includes(shown[0]);
  const days = opening
    ? [...new Set(shown![0].showings.map((s) => new Date(s.startsAt).toDateString()))].map((d) => new Date(d))
    : upcomingDays();
  // The day lives in the URL (?day=1), so refresh, Back and shared links keep it.
  const [params, setParams] = useSearchParams();
  const day = Math.max(0, Math.min(Math.trunc(Number(params.get("day"))) || 0, days.length - 1));
  const setParam = (key: string, value: string) =>
    setParams((p) => (value ? p.set(key, value) : p.delete(key), p), { replace: true });
  const onDay = (i: number) => setParam("day", i ? String(i) : "");
  // A film's times are for one cinema at a time: the one in the URL, else the last one chosen that shows it.
  const cinemas = [...new Map(shown?.[0]?.showings.map((s) => [s.cinemaID, s.cinema]))]
    .sort((a, b) => a[1].localeCompare(b[1]));
  const ids = cinemas.map(([id]) => id);
  const cinema = [params.get("cinema"), ...savedCinemas()].find((id) => id && ids.includes(id)) ?? "";
  function onCinema(id: string) {
    setParam("cinema", id);
    if (id) localStorage.setItem("cinemas", [id, ...savedCinemas().filter((c) => c !== id)].slice(0, MAX_CINEMAS).join(","));
  }
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
        {movieSlug && cinemas.length > 0 && (
          <div className="fields">
            <label className="plain">
              <span>Cinema</span>
              <select value={cinema} onChange={(e) => onCinema(e.target.value)}>
                <option value="" disabled>Select a cinema</option>
                {cinemas.map(([id, name]) => (
                  <option key={id} value={id}>{name}</option>
                ))}
              </select>
            </label>
          </div>
        )}
        {movieSlug && cinema && (
          <nav className="days" aria-label="Day">
            {days.map((d, i) => (
              <button key={i} aria-pressed={i === day} onClick={() => onDay(i)}>
                {opening ? formatDay(d) : i === 0 ? "Today" : i === 1 ? "Tomorrow" : formatDay(d)}
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
              (s) => s.start > now && s.cinemaID === cinema,
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
                  {!movieSlug ? null : !cinemas.length ? (
                    <p className="none">No sessions in the next few days.</p>
                  ) : !cinema ? (
                    <p className="none">Select a cinema to see session times.</p>
                  ) : sessions.length ? (
                    <div className="times">
                      {sessions.map((s) => (
                        <Link
                          key={s.id}
                          to={`/sessions/${s.id}`}
                          state={{ pushed: true }}
                          aria-label={`${movie.title} at ${formatTime(s.start)}, ${s.format}`}
                        >
                          {formatTime(s.start)}
                          {s.format !== "Standard" && <small> {s.format}</small>}
                        </Link>
                      ))}
                    </div>
                  ) : (
                    <p className="none">
                      {day === 0 && !opening
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
                            {day === 0 && !opening ? "tomorrow" : formatDay(days[day + 1])}
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
