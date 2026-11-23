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
  type Room,
} from "@/lib/data";
import { useTitle } from "@/lib/routes";

// Staff manage films and showings here. Screens are set up with SQL. onChanged refreshes the public listings.
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
  const [data, setData] = useState<{ movies: AdminMovie[]; rooms: Room[] } | null>();
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    Promise.all([api.get<AdminMovie[]>("/admin/movies"), api.get<Room[]>("/admin/rooms")]).then(
      ([movies, rooms]) => setData({ movies, rooms }),
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
          data.movies.map((m) => <Film key={m.movieID} movie={m} rooms={data.rooms} onSaved={saved} />)
        ) : (
          <p className="none">No films yet.</p>
        )}
      </section>
    </>
  );
}

function Film({ movie, rooms, onSaved }: { movie: AdminMovie; rooms: Room[]; onSaved: () => void }) {
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
              <Showing key={s.eventID} movieID={movie.movieID} showing={s} rooms={rooms} onSaved={onSaved} />
            ))}
          </ul>
        ) : (
          <p className="none">None.</p>
        )}
        <h3>Add a showing</h3>
        <ShowingForm movieID={movie.movieID} rooms={rooms} onSaved={onSaved} />
      </div>
    </details>
  );
}

function Showing({
  movieID,
  showing,
  rooms,
  onSaved,
}: {
  movieID: string;
  showing: AdminShowing;
  rooms: Room[];
  onSaved: () => void;
}) {
  const [editing, setEditing] = useState(false);
  const start = new Date(showing.startsAt);
  const remove = useDelete(`/admin/events/${showing.eventID}`, `Delete the ${formatTime(start)} showing?`, onSaved);

  return (
    <li>
      <p>
        {formatDay(start)}, {formatTime(start)} · {showing.room} · {money.format(showing.priceCents / 100)} ·{" "}
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
      {editing && <ShowingForm movieID={movieID} showing={showing} rooms={rooms} onSaved={onSaved} />}
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
  onSaved,
}: {
  movieID: string;
  showing?: AdminShowing;
  rooms: Room[];
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
            priceCents: Math.round(Number(f.price) * 100),
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
      <Field
        label="Price per seat ($)"
        name="price"
        type="number"
        required
        min={0.5}
        max={1000}
        step={0.01}
        defaultValue={showing && (showing.priceCents / 100).toFixed(2)}
      />
      <Status error={error} notice={notice} />
      <button className="primary" disabled={busy}>
        {busy ? "Saving…" : showing ? "Save showing" : "Add showing"}
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
