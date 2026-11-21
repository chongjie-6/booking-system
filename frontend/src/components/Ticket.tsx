import { formatDay, formatTime, seatLabel, total, type Seat, type Showtime } from "@/lib/data";

export function Ticket({
  showtime,
  seats,
  bookingRef,
}: {
  showtime: Showtime;
  seats: Seat[];
  bookingRef?: string;
}) {
  return (
    <article className="ticket" aria-label="Ticket">
      <div className="ticket-main">
        <p className="house">Lantern Picture House</p>
        <h2>{showtime.movie.title}</h2>
        <dl>
          <dt>When</dt>
          <dd>
            {formatDay(showtime.start)}, {formatTime(showtime.start)}
          </dd>
          <dt>Where</dt>
          <dd>{showtime.room}</dd>
          <dt>Seats</dt>
          <dd>{seats.map(seatLabel).join(", ")}</dd>
          <dt>Total</dt>
          <dd>{total(showtime, seats.length)}</dd>
        </dl>
      </div>
      <div className="stub">
        <span>Admit</span>
        <strong>{seats.length}</strong>
        {bookingRef && (
          <>
            <span>Ref</span>
            <b className="ref">{bookingRef}</b>
          </>
        )}
      </div>
    </article>
  );
}
