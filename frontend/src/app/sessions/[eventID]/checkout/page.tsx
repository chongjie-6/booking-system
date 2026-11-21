import { useEffect, useState } from "react";
import { api, ApiError, type User } from "@/lib/api";
import { Auth } from "@/components/Auth";
import type { Seat, Showtime } from "@/lib/data";
import { Ticket } from "@/components/Ticket";

// The hold timer warns this long before the seats are released. WCAG 2.2.1 asks for at least 20 seconds.
const HOLD_WARNING_SECONDS = 120;

export default function Checkout({
  showtime,
  seats,
  user,
  onSignedIn,
  onBack,
  onFilms,
  onBooked,
}: {
  showtime: Showtime;
  seats: Seat[];
  user: User | null | undefined;
  onSignedIn: (u: User) => void;
  onBack: () => void;
  onFilms: () => void;
  onBooked: (u: User, ref: string) => void;
}) {
  const [error, setError] = useState<Error | null>(null);
  const [busy, setBusy] = useState(false);
  const [heldUntil, setHeldUntil] = useState<number>();
  // Bumped to hold the seats again, e.g. when the user asks for more time.
  const [holds, setHolds] = useState(0);
  const path = `/events/${showtime.id}`;
  const status = error instanceof ApiError ? error.status : 0;

  // Hold the seats while the user checks out, once they're signed in.
  useEffect(() => {
    if (!user) return;
    const seatIDs = seats.map((s) => s.seatID);
    api.put<{ holdSeconds: number }>(`/events/${showtime.id}/hold`, { seatIDs }).then((r) => {
      setHeldUntil(Date.now() + r.holdSeconds * 1000);
      setError(null);
    }, setError);
  }, [user, showtime.id, seats, holds]);

  async function confirm(u: User) {
    setBusy(true);
    setError(null);
    try {
      const body = { seatIDs: seats.map((s) => s.seatID) };
      const { ref } = await api.post<{ ref: string }>(`${path}/bookings`, body);
      onBooked(u, ref);
    } catch (err) {
      setError(err as Error);
      setBusy(false);
    }
  }

  return (
    <main>
      <button className="back" onClick={onBack} disabled={busy}>
        Change seats
      </button>
      <h1 tabIndex={-1}>Confirm your booking</h1>
      <div className="checkout">
        {user ? (
          <div className="fields">
            <p>
              Booking as {user.firstName} {user.lastName}. We'll send your
              tickets to {user.email}.
            </p>
            {heldUntil && !error && (
              <HoldTimer
                until={heldUntil}
                onExtend={() => setHolds((h) => h + 1)}
              />
            )}
            {error && (
              <p className="error" role="alert">
                {error.message}
              </p>
            )}
            {status === 409 ? (
              <button className="primary" onClick={onBack}>
                Choose other seats
              </button>
            ) : status === 404 ? (
              <button className="primary" onClick={onFilms}>
                Choose another session
              </button>
            ) : (
              <button
                className="primary"
                onClick={() => confirm(user)}
                disabled={busy}
              >
                {busy ? "Booking…" : error ? "Try again" : "Confirm booking"}
              </button>
            )}
          </div>
        ) : user === null ? (
          <Auth onSignedIn={onSignedIn} />
        ) : (
          <p className="none" role="status">
            Loading…
          </p>
        )}
        <Ticket showtime={showtime} seats={seats} />
      </div>
    </main>
  );
}

// Counts down the seat hold. Only the warning is a live region, so screen readers hear it once, not every second.
function HoldTimer({
  until,
  onExtend,
}: {
  until: number;
  onExtend: () => void;
}) {
  const [now, setNow] = useState(Date.now);

  // Tick just after each second boundary of `until`, so the clock never skips or repeats a second.
  useEffect(() => {
    let timer: ReturnType<typeof setTimeout>;
    const schedule = () => {
      const ms = (((until - Date.now()) % 1000) + 1000) % 1000;
      timer = setTimeout(() => {
        const t = Date.now();
        setNow(t);
        if (t < until) schedule();
      }, ms + 20);
    };
    schedule();
    return () => clearTimeout(timer);
  }, [until]);

  const left = Math.max(0, Math.ceil((until - now) / 1000));
  const clock = `${Math.floor(left / 60)}:${String(left % 60).padStart(2, "0")}`;

  return (
    <div className={left <= HOLD_WARNING_SECONDS ? "hold warn" : "hold"}>
      <p>
        {left ? (
          <>
            Seats held for <strong>{clock}</strong>
          </>
        ) : (
          "Your seats are no longer held."
        )}
      </p>
      {left <= HOLD_WARNING_SECONDS && (
        <>
          <p role="alert">
            {left
              ? "Your seats will be released soon."
              : "Someone else could book them now."}
          </p>
          <button className="link" onClick={onExtend}>
            {left ? "I need more time" : "Hold my seats again"}
          </button>
        </>
      )}
    </div>
  );
}
