import { useLocation, useParams } from "react-router";
import { Ticket } from "@/components/Ticket";
import { useTitle, type Entry } from "@/lib/routes";

// booking is missing when the page is opened from a bookmark, since only this tab's history holds it.
export default function Done({ onAgain }: { onAgain: () => void }) {
  const { ref } = useParams();
  const booking = (useLocation().state as Entry | null)?.booking;
  useTitle("Booking confirmed");

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
      ) : (
        <p>
          Your tickets were emailed to you. Show reference {ref} at the door.
        </p>
      )}
      <button className="primary" onClick={onAgain}>
        Book another film
      </button>
    </main>
  );
}
