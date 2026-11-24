import { useEffect, useState } from "react";
import { api, type User } from "@/lib/api";
import { Auth, Field } from "@/components/Auth";
import { Status } from "@/components/Status";
import { useForm } from "@/lib/form";
import {
  formatDay,
  formatTime,
  money,
  toLocalInput,
  type AdminMovie,
  type AdminShowing,
  type Format,
  type Room,
  type Surcharge,
} from "@/lib/data";
import { useTitle } from "@/lib/routes";

// Staff manage films, showings, formats and seat surcharges here. Screens are set up with SQL. onChanged refreshes the public listings.
export default function Admin({
  user,
  onSignedIn,
  onChanged,
}: {
  user: User | null | undefined;
  onSignedIn: (u: User) => void;
  onChanged: () => void;
}) {
  useTitle("Films and showings");

  return (
    <main className="admin">
      <h1 tabIndex={-1}>Films and showings</h1>
      {user?.userRole === "admin" ? (
        <Manage onChanged={onChanged} />
      ) : user ? (
        <p className="none">This page is for cinema staff.</p>
      ) : user === null ? (
        <Auth onSignedIn={onSignedIn} />
      ) : (
        <p className="none" role="status">
          Loading…
        </p>
      )}
    </main>
  );
}

function Manage({ onChanged }: { onChanged: () => void }) {
  // undefined while loading, null if it failed
  const [data, setData] = useState<{
    movies: AdminMovie[];
    rooms: Room[];
    formats: Format[];
    surcharges: Surcharge[];
  } | null>();
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    Promise.all([
      api.get<AdminMovie[]>("/admin/movies"),
      api.get<Room[]>("/admin/rooms"),
      api.get<Format[]>("/admin/formats"),
      api.get<Surcharge[]>("/admin/surcharges"),
    ]).then(
      ([movies, rooms, formats, surcharges]) => setData({ movies, rooms, formats, surcharges }),
      () => setData(null),
    );
  }, [attempt]);

  function saved() {
    setAttempt((a) => a + 1);
    onChanged();
  }

  if (data === undefined)
    return (
      <p className="none" role="status">
        Loading films…
      </p>
    );
  if (data === null)
    return (
      <p className="error" role="alert">
        Couldn't load the films.{" "}
        <button className="link" onClick={() => setAttempt((a) => a + 1)}>
          Try again
        </button>
      </p>
    );

  return (
    <>
      <section>
        <h2>Add a film</h2>
        <MovieForm onSaved={saved} />
      </section>
      <section>
        <h2>Films</h2>
        {data.movies.length ? (
          data.movies.map((m) => <Film key={m.movieID} movie={m} rooms={data.rooms} formats={data.formats} onSaved={saved} />)
        ) : (
          <p className="none">No films yet.</p>
        )}
      </section>
      <section>
        <h2>Formats and prices</h2>
        <p className="rule">A seat costs its showing's format price plus its seat type's surcharge. Bookings keep what they paid.</p>
        <ul className="showings">
          {data.formats.map((f) => (
            <FormatItem key={f.formatID} format={f} onSaved={saved} />
          ))}
        </ul>
        <h3>Add a format</h3>
        <FormatForm onSaved={saved} />
      </section>
      <section>
        <h2>Seat surcharges</h2>
        <SurchargeForm surcharges={data.surcharges} onSaved={saved} />
      </section>
    </>
  );
}

function Film({
  movie,
  rooms,
  formats,
  onSaved,
}: {
  movie: AdminMovie;
  rooms: Room[];
  formats: Format[];
  onSaved: () => void;
}) {
  const remove = useDelete(`/admin/movies/${movie.movieID}`, `Delete ${movie.title}?`, onSaved);
  const n = movie.showings.length;

  return (
    <details className="admin-film">
      <summary>
        <strong>{movie.title}</strong> {movie.rating} · {n} upcoming showing{n === 1 ? "" : "s"}
        {movie.featured && " · featured"}
      </summary>
      <div className="admin-film-body">
        <MovieForm movie={movie} onSaved={onSaved} />
        <div className="row-actions">
          <button className="link" onClick={remove.run} disabled={remove.busy}>
            Delete film
          </button>
          <Status error={remove.error} notice="" />
        </div>
        <h3>Upcoming showings</h3>
        {n ? (
          <ul className="showings">
            {movie.showings.map((s) => (
              <Showing key={s.eventID} movieID={movie.movieID} showing={s} rooms={rooms} formats={formats} onSaved={onSaved} />
            ))}
          </ul>
        ) : (
          <p className="none">None.</p>
        )}
        <h3>Add a showing</h3>
        <ShowingForm movieID={movie.movieID} rooms={rooms} formats={formats} onSaved={onSaved} />
      </div>
    </details>
  );
}

function Showing({
  movieID,
  showing,
  rooms,
  formats,
  onSaved,
}: {
  movieID: string;
  showing: AdminShowing;
  rooms: Room[];
  formats: Format[];
  onSaved: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const start = new Date(showing.startsAt);
  const remove = useDelete(`/admin/events/${showing.eventID}`, `Delete the ${formatTime(start)} showing?`, onSaved);

  return (
    <li>
      <p>
        {formatDay(start)}, {formatTime(start)} · {showing.room} · {showing.format} ·{" "}
        {showing.booked} seat{showing.booked === 1 ? "" : "s"} sold
      </p>
      {showing.booked ? (
        <p className="rule">Sold seats lock a showing, so it can't be changed or deleted.</p>
      ) : (
        <div className="row-actions">
          <button className="link" onClick={() => setEditing(!editing)} aria-expanded={editing}>
            {editing ? "Stop editing" : "Edit"}
          </button>
          <button className="link" onClick={remove.run} disabled={remove.busy}>
            Delete
          </button>
          <Status error={remove.error} notice="" />
        </div>
      )}
      {editing && <ShowingForm movieID={movieID} showing={showing} rooms={rooms} formats={formats} onSaved={onSaved} />}
    </li>
  );
}

function MovieForm({ movie, onSaved }: { movie?: AdminMovie; onSaved: () => void }) {
  const { busy, error, notice, run } = useForm({});

  return (
    <form
      className="fields"
      onSubmit={(e) => {
        const form = e.currentTarget;
        run(e, async (f) => {
          const body = {
            title: f.title.trim(),
            genre: f.genre.trim(),
            runtimeMinutes: Number(f.runtimeMinutes),
            rating: f.rating.trim(),
            synopsis: f.synopsis.trim(),
            tmdbID: f.tmdbID ? Number(f.tmdbID) : null,
            featured: f.featured === "on",
          };
          if (movie) await api.put(`/admin/movies/${movie.movieID}`, body);
          else {
            await api.post("/admin/movies", body);
            form.reset();
          }
          onSaved();
          return movie ? "Film saved." : `${body.title} added.`;
        });
      }}
    >
      <div className="pair">
        <Field label="Title" name="title" required pattern=".*\S.*" maxLength={200} defaultValue={movie?.title} />
        <Field label="Genre" name="genre" required pattern=".*\S.*" maxLength={100} defaultValue={movie?.genre} />
      </div>
      <div className="pair">
        <Field label="Runtime (minutes)" name="runtimeMinutes" type="number" required min={1} max={600} defaultValue={movie?.runtimeMinutes} />
        <Field label="Rating, e.g. M" name="rating" required pattern=".*\S.*" maxLength={10} defaultValue={movie?.rating} />
      </div>
      <div>
        <Field label="TMDB ID (optional)" name="tmdbID" type="number" min={1} defaultValue={movie?.tmdbID ?? undefined} />
        <p className="rule">With a TMDB ID, the poster, trailer, synopsis and runtime come from TMDB within a minute.</p>
      </div>
      <label className="plain">
        <span>Synopsis</span>
        <textarea name="synopsis" rows={3} maxLength={2000} defaultValue={movie?.synopsis} />
      </label>
      <label className="check">
        <input type="checkbox" name="featured" defaultChecked={movie?.featured} /> Feature in the home page banner
      </label>
      <Status error={error} notice={notice} />
      <button className="primary" disabled={busy}>
        {busy ? "Saving…" : movie ? "Save film" : "Add film"}
      </button>
    </form>
  );
}

function ShowingForm({
  movieID,
  showing,
  rooms,
  formats,
  onSaved,
}: {
  movieID: string;
  showing?: AdminShowing;
  rooms: Room[];
  formats: Format[];
  onSaved: () => void;
}) {
  const { busy, error, notice, run } = useForm({});

  return (
    <form
      className="fields"
      onSubmit={(e) => {
        const form = e.currentTarget;
        run(e, async (f) => {
          const body = {
            movieID,
            roomID: f.roomID,
            startsAt: new Date(f.startsAt).toISOString(), // the input's value is in the browser's time zone
            formatID: f.formatID,
          };
          if (showing) await api.put(`/admin/events/${showing.eventID}`, body);
          else {
            await api.post("/admin/events", body);
            form.reset();
          }
          onSaved();
          return showing ? "Showing saved." : "Showing added.";
        });
      }}
    >
      <div className="pair">
        <label className="plain">
          <span>Screen</span>
          <select name="roomID" required defaultValue={showing?.roomID ?? ""}>
            <option value="" disabled>
              Choose a screen
            </option>
            {rooms.map((r) => (
              <option key={r.roomID} value={r.roomID}>
                {r.name} ({r.capacity} seats)
              </option>
            ))}
          </select>
        </label>
        <label className="plain">
          <span>Starts</span>
          <input type="datetime-local" name="startsAt" required defaultValue={showing && toLocalInput(showing.startsAt)} />
        </label>
      </div>
      <label className="plain">
        <span>Format</span>
        <select name="formatID" required defaultValue={showing?.formatID ?? ""}>
          <option value="" disabled>
            Choose a format
          </option>
          {formats.map((f) => (
            <option key={f.formatID} value={f.formatID}>
              {f.name} ({money.format(f.basePriceCents / 100)} a seat)
            </option>
          ))}
        </select>
      </label>
      <Status error={error} notice={notice} />
      <button className="primary" disabled={busy}>
        {busy ? "Saving…" : showing ? "Save showing" : "Add showing"}
      </button>
    </form>
  );
}

function FormatItem({ format, onSaved }: { format: Format; onSaved: () => void }) {
  const [editing, setEditing] = useState(false);
  const remove = useDelete(`/admin/formats/${format.formatID}`, `Delete the ${format.name} format?`, onSaved);

  return (
    <li>
      <p>
        {format.name} · {money.format(format.basePriceCents / 100)} a seat
      </p>
      <div className="row-actions">
        <button className="link" onClick={() => setEditing(!editing)} aria-expanded={editing}>
          {editing ? "Stop editing" : "Edit"}
        </button>
        <button className="link" onClick={remove.run} disabled={remove.busy}>
          Delete
        </button>
        <Status error={remove.error} notice="" />
      </div>
      {editing && <FormatForm format={format} onSaved={onSaved} />}
    </li>
  );
}

function FormatForm({ format, onSaved }: { format?: Format; onSaved: () => void }) {
  const { busy, error, notice, run } = useForm({});

  return (
    <form
      className="fields"
      onSubmit={(e) => {
        const form = e.currentTarget;
        run(e, async (f) => {
          const body = { name: f.name.trim(), basePriceCents: Math.round(Number(f.price) * 100) };
          if (format) await api.put(`/admin/formats/${format.formatID}`, body);
          else {
            await api.post("/admin/formats", body);
            form.reset();
          }
          onSaved();
          return format ? "Format saved." : `${body.name} added.`;
        });
      }}
    >
      <div className="pair">
        <Field label="Name, e.g. IMAX" name="name" required pattern=".*\S.*" maxLength={50} defaultValue={format?.name} />
        <Field
          label="Price per seat ($)"
          name="price"
          type="number"
          required
          min={0.5}
          max={1000}
          step={0.01}
          defaultValue={format && (format.basePriceCents / 100).toFixed(2)}
        />
      </div>
      <Status error={error} notice={notice} />
      <button className="primary" disabled={busy}>
        {busy ? "Saving…" : format ? "Save format" : "Add format"}
      </button>
    </form>
  );
}

// One form for every seat type's surcharge; saving sends each of them.
function SurchargeForm({ surcharges, onSaved }: { surcharges: Surcharge[]; onSaved: () => void }) {
  const { busy, error, notice, run } = useForm({});

  return (
    <form
      className="fields"
      onSubmit={(e) =>
        run(e, async (f) => {
          await Promise.all(
            surcharges.map((s) =>
              api.put(`/admin/surcharges/${s.seatType}`, { surchargeCents: Math.round(Number(f[s.seatType]) * 100) }),
            ),
          );
          onSaved();
          return "Surcharges saved.";
        })
      }
    >
      <p className="rule">Added to the format's price for each seat of that type.</p>
      <div className="pair">
        {surcharges.map((s) => (
          <Field
            key={s.seatType}
            label={`${s.seatType[0].toUpperCase()}${s.seatType.slice(1)} seats (+$)`}
            name={s.seatType}
            type="number"
            required
            min={0}
            max={1000}
            step={0.01}
            defaultValue={(s.surchargeCents / 100).toFixed(2)}
          />
        ))}
      </div>
      <Status error={error} notice={notice} />
      <button className="primary" disabled={busy}>
        {busy ? "Saving…" : "Save surcharges"}
      </button>
    </form>
  );
}

// Deletes path after the user confirms, then calls onDeleted. Returns run for a button, with busy and error states.
function useDelete(path: string, question: string, onDeleted: () => void) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function run() {
    if (!confirm(question)) return;
    setBusy(true);
    setError("");
    try {
      await api.del(path);
      onDeleted();
    } catch (err) {
      setError((err as Error).message);
      setBusy(false);
    }
  }

  return { busy, error, run };
}
