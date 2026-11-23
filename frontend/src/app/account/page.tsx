import { useEffect, useId, useState } from "react";
import { Link } from "react-router";
import { api, type User } from "@/lib/api";
import { Auth, Field } from "@/components/Auth";
import { Status } from "@/components/Status";
import { useForm } from "@/lib/form";
import { Ticket } from "@/components/Ticket";
import { canCancel, splitBookings, ticketShowtime, type UserBooking } from "@/lib/data";
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
  // now is when they loaded, which decides what's upcoming and what can still be cancelled.
  const [bookings, setBookings] = useState<(ReturnType<typeof splitBookings> & { now: number }) | null>();
  const [attempt, setAttempt] = useState(0);
  const [cancelling, setCancelling] = useState("");
  const [cancelError, setCancelError] = useState("");
  const [cancelled, setCancelled] = useState("");

  useEffect(() => {
    api.get<UserBooking[]>("/bookings").then(
      (b) => {
        const now = Date.now();
        setBookings({ ...splitBookings(b ?? [], now), now });
      },
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

  const { upcoming, past, now } = bookings;

  async function cancel(b: UserBooking) {
    if (!confirm(`Cancel your booking for ${b.title}? Your seats will be freed and you'll get a full refund.`)) return;
    setCancelling(b.bookingID);
    setCancelError("");
    setCancelled("");
    try {
      await api.del(`/bookings/${b.bookingID}`);
      setCancelled(`Your booking for ${b.title} is cancelled. The refund takes 5 to 10 days to reach your card.`);
      setAttempt((a) => a + 1);
    } catch (err) {
      setCancelError((err as Error).message);
    } finally {
      setCancelling("");
    }
  }

  return (
    <>
      <section>
        <h2>Upcoming</h2>
        <Status error={cancelError} notice={cancelled} />
        {upcoming.length ? (
          <BookingList bookings={upcoming} now={now} cancelling={cancelling} onCancel={cancel} />
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
          <BookingList bookings={past} now={now} past />
        </section>
      )}
    </>
  );
}

function BookingList({
  bookings,
  now,
  past,
  cancelling,
  onCancel,
}: {
  bookings: UserBooking[];
  now: number;
  past?: boolean;
  cancelling?: string;
  onCancel?: (b: UserBooking) => void;
}) {
  return (
    <ul className={past ? "bookings past" : "bookings"}>
      {bookings.map((b) => (
        <li key={b.bookingID}>
          <Ticket showtime={ticketShowtime(b)} seats={b.seats} bookingRef={b.ref} />
          {onCancel && canCancel(b, now) && (
            <button className="link" onClick={() => onCancel(b)} disabled={!!cancelling}>
              {cancelling === b.bookingID ? "Cancelling…" : `Cancel booking ${b.ref}`}
            </button>
          )}
        </li>
      ))}
    </ul>
  );
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
