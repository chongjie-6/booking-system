import { useEffect, useRef, useState } from "react";
import { Link } from "react-router";
import { loadStripe, type Stripe, type StripeElements, type StripePaymentElement } from "@stripe/stripe-js";
import { api, ApiError, type User } from "@/lib/api";
import { Auth } from "@/components/Auth";
import { total, totalCents, type Seat, type Showtime } from "@/lib/data";
import { Steps } from "@/components/Steps";
import { Ticket } from "@/components/Ticket";

// The hold timer warns this long before the seats are released. WCAG 2.2.1 asks for at least 20 seconds.
const HOLD_WARNING_SECONDS = 120;

type Payment = { clientSecret: string; publishableKey: string; amountCents: number };

// Stripe.js loads from Stripe on first use, and only once.
let stripeJS: Promise<Stripe | null> | undefined;

// Stripe's card form is an iframe, so it gets the site's colours and font this way rather than from the stylesheet.
const appearance = {
  theme: "night" as const,
  variables: {
    colorPrimary: "#e8b64c",
    colorBackground: "#3a0f1b",
    colorText: "#f4e7d0",
    colorDanger: "#f59aa6",
    fontFamily: "Figtree, system-ui, sans-serif",
    borderRadius: "6px",
  },
};
const fonts = [{ cssSrc: "https://fonts.googleapis.com/css2?family=Figtree:wght@400;600&display=swap" }];

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
  const [card, setCard] = useState<{ stripe: Stripe; elements: StripeElements }>();
  // Kept apart from error, so a hold that succeeds afterwards doesn't clear it.
  const [startError, setStartError] = useState<Error | null>(null);
  // Bumped to start the payment again after it failed to start.
  const [starts, setStarts] = useState(0);
  // Paid, but the booking didn't come back. Stripe's webhook makes it, so it shows up in the account soon.
  const [paid, setPaid] = useState(false);
  const box = useRef<HTMLDivElement>(null);
  const path = `/events/${showtime.id}`;
  const shown = error ?? startError;
  const status = shown instanceof ApiError ? shown.status : 0;

  // Hold the seats while the user checks out, once they're signed in.
  useEffect(() => {
    if (!user) return;
    const seatIDs = seats.map((s) => s.seatID);
    api.put<{ holdSeconds: number }>(`/events/${showtime.id}/hold`, { seatIDs }).then((r) => {
      setHeldUntil(Date.now() + r.holdSeconds * 1000);
      setError(null);
    }, setError);
  }, [user, showtime.id, seats, holds]);

  // Start a payment for the seats and show Stripe's card form for it.
  useEffect(() => {
    if (!user) return;
    let gone = false;
    let element: StripePaymentElement | undefined;
    const seatIDs = seats.map((s) => s.seatID);
    api
      .post<Payment>(`${path}/payment`, { seatIDs })
      .then(async (p) => {
        const stripe = await (stripeJS ??= loadStripe(p.publishableKey));
        if (!stripe) throw new Error("Couldn't load the card form. Check your connection and try again.");
        if (gone || !box.current) return;
        const elements = stripe.elements({ clientSecret: p.clientSecret, appearance, fonts });
        element = elements.create("payment");
        element.mount(box.current);
        setCard({ stripe, elements });
      })
      .catch((err) => {
        stripeJS = undefined; // so a retry loads Stripe.js again if that's what failed
        if (!gone) setStartError(err);
      });
    return () => {
      gone = true;
      element?.destroy();
      setCard(undefined);
    };
  }, [user, path, seats, starts]);

  async function pay(u: User) {
    if (!card) return;
    setBusy(true);
    setError(null);
    const result = await card.stripe.confirmPayment({ elements: card.elements, redirect: "if_required" });
    if (result.error) {
      setError(new Error(result.error.message ?? "Your payment didn't go through. Please try again."));
      setBusy(false);
      return;
    }
    try {
      const { ref } = await api.post<{ ref: string }>(`${path}/bookings`, { paymentIntentID: result.paymentIntent.id });
      onBooked(u, ref);
    } catch (err) {
      // 409 and 404 mean the booking couldn't be made and the payment was refunded; anything else, it's still coming.
      const s = err instanceof ApiError ? err.status : 0;
      if (s === 409 || s === 404) setError(err as Error);
      else setPaid(true);
      setBusy(false);
    }
  }

  function retry(u: User) {
    if (startError) {
      setStartError(null);
      setStarts((n) => n + 1);
    } else pay(u);
  }

  return (
    <main>
      <button className="back" onClick={onBack} disabled={busy}>
        Change seats
      </button>
      <Steps at={1} />
      <h1 tabIndex={-1}>Confirm your booking</h1>
      <div className="checkout">
        {user && paid ? (
          <div className="fields">
            <p role="status">
              Your payment went through. We're finishing your booking and will
              email your tickets to {user.email}.
            </p>
            <Link className="link" to="/account">
              See your bookings
            </Link>
          </div>
        ) : user ? (
          <div className="fields">
            <p>
              Booking as {user.firstName} {user.lastName}. We'll send your
              tickets to {user.email}.
            </p>
            <div className="card" ref={box} />
            <p className="rule">
              Cancel from your account up to an hour before the session for a
              full refund.
            </p>
            {!card && !shown && (
              <p className="none" role="status">
                Loading the card form…
              </p>
            )}
            {heldUntil && !shown && (
              <HoldTimer
                until={heldUntil}
                onExtend={() => setHolds((h) => h + 1)}
              />
            )}
            {shown && (
              <p className="error" role="alert">
                {shown.message}
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
                onClick={() => (shown ? retry(user) : pay(user))}
                disabled={busy || (!card && !shown)}
              >
                {busy ? "Paying…" : shown ? "Try again" : `Pay ${total(seats)}`}
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
        <Ticket showtime={showtime} seats={seats} totalCents={totalCents(seats)} />
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
