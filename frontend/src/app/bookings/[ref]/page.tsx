import { useEffect, useState } from "react";
import { Link, useLocation, useParams } from "react-router";
import { api } from "@/lib/api";
import { Ticket } from "@/components/Ticket";
import { ticketShowtime, type UserBooking } from "@/lib/data";
import { useTitle, type Entry } from "@/lib/routes";

// Straight after booking, this tab's history holds the booking. Opened from a link, it's fetched; signed out, it can't be.
export default function Done({ onAgain }: { onAgain: () => void }) {
  const { ref = "" } = useParams();
  const booking = (useLocation().state as Entry | null)?.booking;
  // undefined while loading, null if it couldn't be fetched
  const [fetched, setFetched] = useState<UserBooking | null>();
  useTitle("Booking confirmed");

  useEffect(() => {
    if (!booking) api.get<UserBooking>(`/bookings/${encodeURIComponent(ref)}`).then(setFetched, () => setFetched(null));
  }, [booking, ref]);

  return (
    <main className="done">
      <h1 tabIndex={-1}>Booking confirmed</h1>
      {booking ? (
        <>
          <p>
            {booking.name}, your tickets are on their way to {booking.email}.
            Show reference {booking.ref} at the door.
          </p>
          <Ticket
            showtime={booking.showtime}
            seats={booking.seats}
            bookingRef={booking.ref}
          />
        </>
      ) : fetched ? (
        <>
          <p>Your tickets were emailed to you. Show reference {fetched.ref} at the door.</p>
          <Ticket showtime={ticketShowtime(fetched)} seats={fetched.seats} bookingRef={fetched.ref} />
        </>
      ) : fetched === null ? (
        <p>
          Your tickets were emailed to you. Show reference {ref} at the door.
        </p>
      ) : (
        <p className="none" role="status">
          Loading your booking…
        </p>
      )}
      <button className="primary" onClick={onAgain}>
        Book another film
      </button>
      <Link className="link" to="/account">
        See all your bookings
      </Link>
    </main>
  );
}
