import { useEffect, useId, useState, type FormEvent } from "react";
import { Link } from "react-router";
import { api, ApiError, type User } from "@/lib/api";
import { Auth, Field } from "@/components/Auth";
import { Ticket } from "@/components/Ticket";
import { splitBookings, type UserBooking } from "@/lib/data";
import { useTitle } from "@/lib/routes";

export default function Account({
  user,
  onSignedIn,
  onUpdated,
}: {
  user: User | null | undefined;
  onSignedIn: (u: User) => void;
  onUpdated: (u: User) => void;
}) {
  useTitle("Your account");

  return (
    <main className="account-page">
      <h1 tabIndex={-1}>Your account</h1>
      {user ? (
        <>
          <Bookings />
          <div className="settings">
            <Details key={user.userID} user={user} onUpdated={onUpdated} />
            <Password />
          </div>
        </>
      ) : user === null ? (
        <section>
          <p>Sign in to see your bookings and change your details.</p>
          <Auth onSignedIn={onSignedIn} />
        </section>
      ) : (
        <p className="none" role="status">
          Loading…
        </p>
      )}
    </main>
  );
}

function Bookings() {
  // undefined while loading, null if it failed
  const [bookings, setBookings] = useState<ReturnType<typeof splitBookings> | null>();
  const [attempt, setAttempt] = useState(0);

  useEffect(() => {
    api.get<UserBooking[]>("/bookings").then(
      (b) => setBookings(splitBookings(b ?? [], Date.now())),
      () => setBookings(null),
    );
  }, [attempt]);

  if (bookings === undefined)
    return (
      <p className="none" role="status">
        Loading your bookings…
      </p>
    );
  if (bookings === null)
    return (
      <p className="error" role="alert">
        Couldn't load your bookings.{" "}
        <button
          className="link"
          onClick={() => {
            setBookings(undefined);
            setAttempt((a) => a + 1);
          }}
        >
          Try again
        </button>
      </p>
    );

  const { upcoming, past } = bookings;

  return (
    <>
      <section>
        <h2>Upcoming</h2>
        {upcoming.length ? (
          <BookingList bookings={upcoming} />
        ) : (
          <p className="none">
            No upcoming bookings.{" "}
            <Link className="link" to="/">
              See what's showing
            </Link>
          </p>
        )}
      </section>
      {past.length > 0 && (
        <section>
          <h2>Past</h2>
          <BookingList bookings={past} past />
        </section>
      )}
    </>
  );
}

function BookingList({ bookings, past }: { bookings: UserBooking[]; past?: boolean }) {
  return (
    <ul className={past ? "bookings past" : "bookings"}>
      {bookings.map((b) => (
        <li key={b.bookingID}>
          <Ticket
            showtime={{
              movie: { title: b.title },
              start: new Date(b.startsAt),
              priceCents: b.priceCents,
              room: b.room,
            }}
            seats={b.seats}
            bookingRef={b.ref}
          />
        </li>
      ))}
    </ul>
  );
}

// Submits a form's fields, tracking busy, error and success states. messages maps a status to friendlier text.
function useForm(messages: Record<number, string>) {
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");

  async function run(e: FormEvent<HTMLFormElement>, send: (f: Record<string, string>) => Promise<string>) {
    e.preventDefault();
    const form = e.currentTarget;
    const f = Object.fromEntries(new FormData(form)) as Record<string, string>;
    setBusy(true);
    setError("");
    setNotice("");
    try {
      setNotice(await send(f));
    } catch (err) {
      setError((err instanceof ApiError && messages[err.status]) || (err as Error).message);
    } finally {
      setBusy(false);
    }
  }

  return { busy, error, notice, run };
}

function Details({ user, onUpdated }: { user: User; onUpdated: (u: User) => void }) {
  const { busy, error, notice, run } = useForm({ 409: "That email is already used by another account." });

  return (
    <form
      className="fields"
      onSubmit={(e) =>
        run(e, async (f) => {
          const body = { firstName: f.firstName.trim(), lastName: f.lastName.trim(), email: f.email.trim() };
          onUpdated(await api.patch<User>("/user", body));
          return "Details saved.";
        })
      }
    >
      <h2>Your details</h2>
      <div className="pair">
        <Field label="First name" name="firstName" required pattern=".*\S.*" defaultValue={user.firstName} autoComplete="given-name" />
        <Field label="Last name" name="lastName" required pattern=".*\S.*" defaultValue={user.lastName} autoComplete="family-name" />
      </div>
      <Field label="Email" name="email" type="email" required defaultValue={user.email} autoComplete="email" />
      <Status error={error} notice={notice} />
      <button className="primary" disabled={busy}>
        {busy ? "Saving…" : "Save details"}
      </button>
    </form>
  );
}

function Password() {
  const { busy, error, notice, run } = useForm({ 403: "Your current password is wrong." });
  const ruleId = useId();

  return (
    <form
      className="fields"
      onSubmit={(e) => {
        const form = e.currentTarget;
        run(e, async (f) => {
          await api.post("/auth/password", { currentPassword: f.currentPassword, newPassword: f.newPassword });
          form.reset();
          return "Password changed. Every other device has been signed out.";
        });
      }}
    >
      <h2>Change password</h2>
      <div className="group">
        <Field label="Current password" name="currentPassword" type="password" required autoComplete="current-password" />
        <div>
          <Field
            label="New password"
            name="newPassword"
            type="password"
            required
            minLength={8}
            maxLength={72}
            autoComplete="new-password"
            aria-describedby={ruleId}
          />
          <p className="rule" id={ruleId}>
            At least 8 characters.
          </p>
        </div>
      </div>
      <Status error={error} notice={notice} />
      <button className="primary" disabled={busy}>
        {busy ? "Saving…" : "Change password"}
      </button>
    </form>
  );
}

function Status({ error, notice }: { error: string; notice: string }) {
  return (
    <>
      {error && (
        <p className="error" role="alert">
          {error}
        </p>
      )}
      {notice && (
        <p className="hint" role="status">
          {notice}
        </p>
      )}
    </>
  );
}
