import { useEffect, useState } from "react";
import { useNavigate } from "react-router";
import { api } from "@/lib/api";
import {
  MAX_CINEMAS,
  TIMES,
  dayKey,
  distanceKm,
  findSessions,
  formatDay,
  formatKm,
  toParams,
  upcomingDays,
  type Cinema,
  type Movie,
  type Place,
  type SessionSearch,
  type TimeOfDay,
} from "@/lib/data";
import { Drawer } from "@/components/Drawer";
import { Poster } from "@/components/Poster";

type Step = "cinema" | "movie" | "date";

const flip = <T,>(list: T[], v: T) => (list.includes(v) ? list.filter((x) => x !== v) : [...list, v]);
const summary = (names: string[], none: string) =>
  !names.length ? none : names.length === 1 ? names[0] : `${names[0]} +${names.length - 1}`;

// Search for sessions by cinema, then optionally film, day and time of day, one step at a time in a modal.
// Find sessions opens /sessions with the search in the URL.
export function QuickBook({ movies, initial }: { movies: Movie[] | null | undefined; initial: SessionSearch }) {
  const navigate = useNavigate();
  const [q, setQ] = useState(initial);
  const [open, setOpen] = useState(false);
  const [step, setStep] = useState<Step>("cinema");
  const [cinemas, setCinemas] = useState<Cinema[]>([]);
  const [region, setRegion] = useState("");
  const [query, setQuery] = useState("");
  // Where the user is, once they ask for cinemas near them. Kept only while the page is open.
  const [here, setHere] = useState<Place | null>(null);
  const [locating, setLocating] = useState<"idle" | "busy" | "blocked" | "failed">("idle");
  const days = upcomingDays();

  useEffect(() => {
    api.get<Cinema[]>("/cinemas").then((list) => {
      setCinemas(list);
      // A saved cinema that has since closed is dropped.
      setQ((q) => ({ ...q, cinemas: q.cinemas.filter((id) => list.some((c) => c.cinemaID === id)) }));
    }, () => {});
  }, []);

  function update(change: Partial<SessionSearch>) {
    setQ({ ...q, ...change });
    if (change.cinemas) localStorage.setItem("cinemas", change.cinemas.join(","));
  }

  function show(s: Step) {
    setStep(s);
    setOpen(true);
  }

  // Asks the browser where the user is, then lists cinemas nearest first, starting on the nearest one's state.
  function locate() {
    setLocating("busy");
    navigator.geolocation.getCurrentPosition(
      ({ coords }) => {
        const place = { lat: coords.latitude, lng: coords.longitude };
        setHere(place);
        setLocating("idle");
        const nearest = cinemas.toSorted((a, b) => distanceKm(place, a) - distanceKm(place, b))[0];
        if (nearest) setRegion(nearest.state);
      },
      (err) => setLocating(err.code === err.PERMISSION_DENIED ? "blocked" : "failed"),
      { maximumAge: 10 * 60_000, timeout: 15_000 },
    );
  }

  function find() {
    setOpen(false);
    navigate(`/sessions?${toParams(q)}`, { state: { pushed: true } });
  }

  const chosenCinemas = cinemas.filter((c) => q.cinemas.includes(c.cinemaID));
  const states = [...new Set(cinemas.map((c) => c.state))];
  const tab = states.includes(region) ? region : (chosenCinemas[0]?.state ?? states[0]);
  // Only films on at the chosen cinemas, in the listing's order (featured first).
  const onAtCinemas = new Set(findSessions(movies ?? [], { ...q, movies: [], days: [], times: [] }, new Date()).map((s) => s.movie));
  const films = (movies ?? []).filter((m) => onAtCinemas.has(m));
  const chosenFilms = films.filter((m) => q.movies.includes(m.movieID));
  const dayName = (d: Date, i: number) => (i === 0 ? "Today" : i === 1 ? "Tomorrow" : formatDay(d));
  const dayNames = days.flatMap((d, i) => (q.days.includes(dayKey(d)) ? [dayName(d, i)] : []));
  const whenNames = [...dayNames, ...q.times.map((t) => t[0].toUpperCase() + t.slice(1))];
  const findLabel = chosenCinemas.length === 1 ? `Find sessions at ${chosenCinemas[0].name}` : "Find sessions";

  return (
    <>
      <div className="quickbook">
        <div className="quickbook-field">
          <button onClick={() => show("cinema")}>
            <small>Cinema</small>
            {summary(chosenCinemas.map((c) => c.name), "Select cinema")}
          </button>
          {!!chosenCinemas.length && (
            <button className="clear" aria-label="Clear cinema" onClick={() => update({ cinemas: [] })}>
              ×
            </button>
          )}
        </div>
        <button onClick={() => show("movie")}>
          <small>Movie</small>
          {summary(chosenFilms.map((m) => m.title), "Select movie")}
        </button>
        <button onClick={() => show("date")}>
          <small>Date</small>
          {summary(whenNames, "Add date")}
        </button>
        <button className="primary" onClick={find}>
          Find sessions
        </button>
      </div>

      <Drawer className="modal" open={open} onClose={() => setOpen(false)}>
        {step === "cinema" && (
          <>
            <h2>Which cinema?</h2>
            <p className="pick-hint">Choose up to {MAX_CINEMAS}.</p>
            {"geolocation" in navigator && (
              <p className="locate">
                <button className="link" onClick={locate} disabled={locating === "busy"}>
                  <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" aria-hidden="true">
                    <path d="M12 21s-7-6.2-7-11.5a7 7 0 0 1 14 0C19 14.8 12 21 12 21z" />
                    <circle cx="12" cy="9.5" r="2.5" />
                  </svg>
                  {locating === "busy" ? "Finding you…" : "Use my location"}
                </button>
                {here && locating === "idle" && <span className="note">Sorted nearest first</span>}
                <span role="status">
                  {locating === "blocked" && "Location is blocked. Allow it in your browser's site settings to see cinemas near you."}
                  {locating === "failed" && "Couldn't find your location. Try again."}
                </span>
              </p>
            )}
            {states.length > 1 && (
              <nav className="days" aria-label="State">
                {states.map((s) => (
                  <button key={s} aria-pressed={s === tab} onClick={() => setRegion(s)}>
                    {s}
                  </button>
                ))}
              </nav>
            )}
            <ul className="pick-list">
              {cinemas
                .filter((c) => c.state === tab)
                .toSorted((a, b) => (here ? distanceKm(here, a) - distanceKm(here, b) : 0))
                .map((c) => {
                  const on = q.cinemas.includes(c.cinemaID);
                  return (
                    <li key={c.cinemaID}>
                      <label>
                        <span>
                          {c.name}
                          <small>
                            {here && <b>{formatKm(distanceKm(here, c))} · </b>}
                            {c.address}, {c.city}
                          </small>
                        </span>
                        <input
                          type="checkbox"
                          checked={on}
                          disabled={!on && q.cinemas.length >= MAX_CINEMAS}
                          onChange={() => update({ cinemas: flip(q.cinemas, c.cinemaID) })}
                        />
                      </label>
                    </li>
                  );
                })}
            </ul>
          </>
        )}

        {step === "movie" && (
          <>
            <h2>What do you want to see?</h2>
            <p className="pick-hint">Choose one or more films, or none to see them all.</p>
            <input
              className="pick-search"
              type="search"
              placeholder="Search films"
              aria-label="Search films"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
            />
            <ul className="pick-posters">
              {films
                .filter((m) => m.title.toLowerCase().includes(query.trim().toLowerCase()))
                .map((m) => (
                  <li key={m.movieID}>
                    <label>
                      <input
                        type="checkbox"
                        checked={q.movies.includes(m.movieID)}
                        onChange={() => update({ movies: flip(q.movies, m.movieID) })}
                      />
                      <Poster movie={m} />
                      <span>{m.title}</span>
                    </label>
                  </li>
                ))}
            </ul>
            {!films.length && <p className="none">Nothing is on at {summary(chosenCinemas.map((c) => c.name), "any cinema")} in the next few days.</p>}
          </>
        )}

        {step === "date" && (
          <>
            <h2>When do you want to go?</h2>
            <p className="pick-hint">Choose one or more days and times.</p>
            <nav className="days" aria-label="Day">
              <button aria-pressed={!q.days.length} onClick={() => update({ days: [] })}>
                Any day
              </button>
              {days.map((d, i) => (
                <button
                  key={i}
                  aria-pressed={q.days.includes(dayKey(d))}
                  onClick={() => update({ days: flip(q.days, dayKey(d)) })}
                >
                  {dayName(d, i)}
                </button>
              ))}
            </nav>
            <ul className="pick-list">
              {(Object.keys(TIMES) as TimeOfDay[]).map((t) => (
                <li key={t}>
                  <label>
                    <span>
                      {t[0].toUpperCase() + t.slice(1)}
                      <small>{TIMES[t]}</small>
                    </span>
                    <input type="checkbox" checked={q.times.includes(t)} onChange={() => update({ times: flip(q.times, t) })} />
                  </label>
                </li>
              ))}
            </ul>
          </>
        )}

        <div className="pick-actions">
          {step === "cinema" && (
            <button className="secondary" onClick={() => setStep("movie")}>
              Add a movie
            </button>
          )}
          {step === "movie" && (
            <button className="secondary" onClick={() => setStep("date")}>
              Add dates
            </button>
          )}
          <button className="primary" onClick={find}>
            {findLabel}
          </button>
        </div>
      </Drawer>
    </>
  );
}
