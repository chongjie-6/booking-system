import { useState, type CSSProperties } from "react";
import {
  backdropURL,
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
import { Drawer } from "@/components/Drawer";
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
    ? movies &&
      soon &&
      [...movies, ...soon].filter((m) => slug(m.title) === movieSlug)
    : movies;
  // A coming-soon film's days are the ones it has showings on; anything else gets today and the next two.
  const opening = !!shown?.[0] && !movies?.includes(shown[0]);
  const days = opening
    ? [
        ...new Set(
          shown![0].showings.map((s) => new Date(s.startsAt).toDateString()),
        ),
      ].map((d) => new Date(d))
    : upcomingDays();
  // The day lives in the URL (?day=1), so refresh, Back and shared links keep it.
  const [params, setParams] = useSearchParams();
  const day = Math.max(
    0,
    Math.min(Math.trunc(Number(params.get("day"))) || 0, days.length - 1),
  );
  const setParam = (key: string, value: string) =>
    setParams((p) => (value ? p.set(key, value) : p.delete(key), p), {
      replace: true,
    });
  const onDay = (i: number) => setParam("day", i ? String(i) : "");
  // A film's times are for the cinemas in the URL (?cinema=a,b), else the saved ones that show it.
  const cinemas = [
    ...new Map(shown?.[0]?.showings.map((s) => [s.cinemaID, s.cinema])),
  ].sort((a, b) => a[1].localeCompare(b[1]));
  const ids = cinemas.map(([id]) => id);
  const chosen = (params.get("cinema")?.split(",") ?? savedCinemas())
    .filter((id) => ids.includes(id))
    .slice(0, MAX_CINEMAS);
  // Saved cinemas this film isn't at stay saved; the ones it is at follow the ticks.
  function onCinema(id: string) {
    const next = chosen.includes(id)
      ? chosen.filter((c) => c !== id)
      : [...chosen, id];
    setParam("cinema", next.join(","));
    localStorage.setItem(
      "cinemas",
      [...next, ...savedCinemas().filter((c) => !ids.includes(c))]
        .slice(0, MAX_CINEMAS)
        .join(","),
    );
  }
  useTitle((movieSlug && shown?.[0]?.title) || "Now showing");

  const movie = movieSlug ? shown?.[0] : undefined;
  const [trailer, setTrailer] = useState(false);
  const [picking, setPicking] = useState(false);
  const showing = cinemas.filter(([id]) => chosen.includes(id));
  const sessions = movie
    ? showtimesFor(movie, days[day]).filter(
        (s) => s.start > now && chosen.includes(s.cinemaID),
      )
    : [];

  return (
    <>
      <main>
        {movieSlug && !movie && (
          <Link className="back" to="/movies">
            All films
          </Link>
        )}
        {movie ? (
          <>
            <header
              className="film-head"
              style={
                movie.backdropPath
                  ? ({
                      "--backdrop": `url(${backdropURL(movie.backdropPath)})`,
                    } as CSSProperties)
                  : undefined
              }
            >
              <Link className="back" to="/movies">
                All films
              </Link>
              <Poster movie={movie} />
              <div>
                <h1 tabIndex={-1}>{movie.title}</h1>
                <p className="rating">{movie.rating}</p>
                <p className="meta">
                  {formatRuntime(movie.runtimeMinutes)} • {movie.genre}
                </p>
              </div>
              <p className="synopsis">{movie.synopsis}</p>
              {movie.trailerKey && (
                <button
                  className="secondary play"
                  onClick={() => setTrailer(true)}
                >
                  Play trailer
                </button>
              )}
            </header>
            <Drawer
              className="modal trailer"
              open={trailer}
              onClose={() => setTrailer(false)}
            >
              {trailer && (
                <iframe
                  src={`https://www.youtube-nocookie.com/embed/${movie.trailerKey}?autoplay=1&rel=0`}
                  title={`${movie.title} trailer`}
                  allow="autoplay; encrypted-media; picture-in-picture; fullscreen"
                />
              )}
            </Drawer>

            <h2 className="times-head">
              {showing.length === 1
                ? `Session times at ${showing[0][1]}`
                : "Session times"}
            </h2>
            {cinemas.length > 0 && (
              <button className="cinema-pick" onClick={() => setPicking(true)}>
                Select cinema{chosen.length ? ` (${chosen.length})` : ""}
              </button>
            )}
            <Drawer
              className="modal"
              open={picking}
              onClose={() => setPicking(false)}
            >
              <h2>Which cinemas?</h2>
              <p className="pick-hint">Choose up to {MAX_CINEMAS}.</p>
              <ul className="pick-list">
                {cinemas.map(([id, name]) => (
                  <li key={id}>
                    <label>
                      <span>{name}</span>
                      <input
                        type="checkbox"
                        checked={chosen.includes(id)}
                        disabled={
                          !chosen.includes(id) && chosen.length >= MAX_CINEMAS
                        }
                        onChange={() => onCinema(id)}
                      />
                    </label>
                  </li>
                ))}
              </ul>
              <div className="pick-actions">
                <button className="primary" onClick={() => setPicking(false)}>
                  Done
                </button>
              </div>
            </Drawer>
            {chosen.length > 0 && (
              <nav className="date-strip" aria-label="Day">
                {days.map((d, i) => {
                  const month =
                    (i === 0 || d.getMonth() !== days[i - 1].getMonth()) &&
                    d.toLocaleDateString("en-AU", { month: "short" });
                  return (
                    <div
                      key={i}
                      className={month && i ? "new-month" : undefined}
                    >
                      <small>{month || " "}</small>
                      <button
                        aria-pressed={i === day}
                        onClick={() => onDay(i)}
                        aria-label={formatDay(d)}
                      >
                        {i === 0 && !opening && <em>Today</em>}
                        <span>
                          {d.toLocaleDateString("en-AU", { weekday: "short" })}
                        </span>
                        <b>{d.getDate()}</b>
                      </button>
                    </div>
                  );
                })}
              </nav>
            )}
            {!cinemas.length ? (
              <p className="none">No sessions in the next few days.</p>
            ) : !chosen.length ? (
              <p className="none">
                Please select one or more cinemas to view sessions.
              </p>
            ) : sessions.length ? (
              showing.map(([id, name]) => {
                const here = sessions.filter((s) => s.cinemaID === id);
                return (
                  <section key={id} className="day-sessions">
                    <h3>{name}</h3>
                    <p className="when">
                      {day === 0 && !opening ? "Today, " : ""}
                      {days[day].toLocaleDateString("en-AU", {
                        weekday: day === 0 && !opening ? undefined : "long",
                        day: "numeric",
                        month: "long",
                      })}
                    </p>
                    {here.length ? (
                      <div className="session-grid">
                        {here.map((s) => (
                          <Link
                            key={s.id}
                            to={`/sessions/${s.id}`}
                            state={{ pushed: true }}
                            data-format={s.format}
                            aria-label={`${movie.title} at ${name}, ${formatTime(s.start)}, ${s.format}`}
                          >
                            <b>{formatTime(s.start)}</b>
                            <small>{s.format}</small>
                          </Link>
                        ))}
                      </div>
                    ) : (
                      <p className="none">No sessions at {name} this day.</p>
                    )}
                  </section>
                );
              })
            ) : (
              <p className="none">
                {day === 0 && !opening
                  ? "No more sessions today."
                  : `No sessions on ${formatDay(days[day])}.`}
                {day < days.length - 1 && (
                  <>
                    {" "}
                    <button className="link" onClick={() => onDay(day + 1)}>
                      See{" "}
                      {day === 0 && !opening
                        ? "tomorrow"
                        : formatDay(days[day + 1])}
                    </button>
                  </>
                )}
              </p>
            )}
          </>
        ) : (
          <h1 tabIndex={-1}>{movieSlug ? "Film" : "Now showing"}</h1>
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

        {!movieSlug && (
          <ul className="listing">
            {movies?.map((movie) => (
              <li key={movie.movieID} className="film">
                <Link
                  to={`/movies/${slug(movie.title)}`}
                  tabIndex={-1}
                  aria-hidden="true"
                >
                  <Poster movie={movie} />
                </Link>
                <div>
                  <h2>
                    <Link to={`/movies/${slug(movie.title)}`}>
                      {movie.title}
                    </Link>
                  </h2>
                  <p className="meta">
                    <span>{movie.genre}</span>
                    <span>{formatRuntime(movie.runtimeMinutes)}</span>
                    <span className="rating">{movie.rating}</span>
                  </p>
                  <p className="synopsis">{movie.synopsis}</p>
                </div>
              </li>
            ))}
          </ul>
        )}
      </main>
      {shown?.some((m) => m.posterPath) && <TmdbCredit />}
    </>
  );
}
