import { useEffect, useState } from "react";
import { useLocation, useNavigate } from "react-router";
import { api, releasing } from "@/lib/api";
import type { Entry } from "@/lib/routes";
import { Poster, TmdbCredit } from "@/components/Poster";
import { Steps } from "@/components/Steps";
import {
  MAX_SEATS,
  formatDay,
  formatRuntime,
  formatTime,
  seatLabel,
  total,
  type Seat,
  type Showtime,
} from "@/lib/data";

export default function SeatPicker({
  showtime,
  onBack,
  onContinue,
}: {
  showtime: Showtime;
  onBack: () => void;
  onContinue: (seats: Seat[]) => void;
}) {
  const location = useLocation();
  const navigate = useNavigate();
  const [seats, onSeats] = useState<Seat[]>(
    () => (location.state as Entry | null)?.seats ?? [],
  );
  // undefined while loading, null if it failed
  const [all, setAll] = useState<Seat[] | null>();
  const [attempt, setAttempt] = useState(0);
  // The seats the user arrived with, e.g. coming back from checkout, and which of those someone else has taken since.
  const [arrivedWith] = useState(seats);
  const [lost, setLost] = useState<Seat[]>([]);
  const atMax = seats.length >= MAX_SEATS;
  const types = new Set(all?.map((s) => s.type));
  const soldOut = !!all?.length && all.every((s) => s.taken);

  useEffect(() => {
    releasing
      .then(() => api.get<Seat[]>(`/events/${showtime.id}/seats`))
      .then(
        (all) => {
          setAll(all);
          const taken = new Set(
            all.filter((s) => s.taken).map((s) => s.seatID),
          );
          setLost(arrivedWith.filter((s) => taken.has(s.seatID)));
          onSeats((seats) => seats.filter((s) => !taken.has(s.seatID)));
        },
        () => setAll(null),
      );
  }, [showtime.id, arrivedWith, attempt]);

  // Keep the seats in this page's history entry, so refresh, and Back from checkout, return with them.
  useEffect(() => {
    const entry = location.state as Entry | null;
    const ids = (seats?: Seat[]) => seats?.map((s) => s.seatID).join();
    if (ids(entry?.seats) !== ids(seats))
      navigate(location, { replace: true, state: { ...entry, seats } });
  }, [location, navigate, seats]);

  // The API sends seats ordered by row then number.
  const rows: Seat[][] = [];
  for (const seat of all ?? []) {
    if (rows.at(-1)?.[0].row === seat.row) rows.at(-1)!.push(seat);
    else rows.push([seat]);
  }

  function toggle(seat: Seat) {
    onSeats((seats) => {
      if (seats.some((s) => s.seatID === seat.seatID))
        return seats.filter((s) => s.seatID !== seat.seatID);
      if (seats.length >= MAX_SEATS) return seats;
      return [...seats, seat].sort(
        (a, b) => a.row.localeCompare(b.row) || a.number - b.number,
      );
    });
  }

  return (
    <>
      <main>
        <button className="back" onClick={onBack}>
          Back to films
        </button>
        <Steps at={0} />
        <header className="session-head">
          <Poster movie={showtime.movie} />
          <div>
            <h1 tabIndex={-1}>{showtime.movie.title}</h1>
            <p className="meta">
              <span>
                {formatDay(showtime.start)}, {formatTime(showtime.start)}
              </span>
              <span>{formatRuntime(showtime.movie.runtimeMinutes)}</span>
              <span>{showtime.room}</span>
            </p>
          </div>
        </header>

        {all === undefined && (
          <p className="none" role="status">
            Loading seats…
          </p>
        )}
        {all === null && (
          <p className="error" role="alert">
            Couldn't load the seats.{" "}
            <button
              className="link"
              onClick={() => {
                setAll(undefined);
                setAttempt((a) => a + 1);
              }}
            >
              Try again
            </button>
          </p>
        )}
        {lost.length > 0 && (
          <p className="hint" role="status">
            Someone else just took {lost.map(seatLabel).join(", ")}, so{" "}
            {lost.length === 1 ? "it's" : "they're"} no longer in your
            selection.
          </p>
        )}
        {soldOut && (
          <p className="none">
            This session is sold out.{" "}
            <button className="link" onClick={onBack}>
              Choose another session
            </button>
          </p>
        )}

        <div className="auditorium">
          <div className="screen" aria-hidden="true" />
          <div className="seats" role="group" aria-label="Seats">
            {rows.map((row) => (
              <div className="row" key={row[0].row}>
                <span className="row-label" aria-hidden="true">
                  {row[0].row}
                </span>
                {row.map((seat) => {
                  const isTaken = seat.taken;
                  const selected = seats.some((s) => s.seatID === seat.seatID);
                  return (
                    <button
                      key={seat.seatID}
                      className="seat"
                      data-type={seat.type}
                      disabled={isTaken}
                      aria-pressed={selected}
                      aria-label={`Row ${seat.row}, seat ${seat.number}${seat.type === "standard" ? "" : `, ${seat.type}`}${isTaken ? ", taken" : ""}`}
                      onClick={() => toggle(seat)}
                    >
                      {selected
                        ? seat.number
                        : seat.type === "wheelchair" && "♿︎"}
                    </button>
                  );
                })}
                <span className="row-label" aria-hidden="true">
                  {row[0].row}
                </span>
              </div>
            ))}
          </div>
        </div>

        {all && (
          <ul className="legend">
            <li>
              <span className="swatch" /> Available
            </li>
            <li>
              <span className="swatch selected" /> Your seats
            </li>
            <li>
              <span className="swatch taken" /> Taken
            </li>
            {types.has("wheelchair") && (
              <li>
                <span className="swatch" data-type="wheelchair">
                  ♿&#xFE0E;
                </span>{" "}
                Wheelchair space
              </li>
            )}
            {types.has("companion") && (
              <li>
                <span className="swatch" data-type="companion" /> Companion seat
              </li>
            )}
            {types.has("premium") && (
              <li>
                <span className="swatch" data-type="premium" /> Premium
              </li>
            )}
          </ul>
        )}

        <footer className="tray">
          <div aria-live="polite">
            <p>
              {seats.length
                ? `${seats.length} ${seats.length === 1 ? "seat" : "seats"}: ${seats.map(seatLabel).join(", ")}`
                : `Choose up to ${MAX_SEATS} seats.`}
            </p>
            {atMax && (
              <p className="hint">
                That's the most seats one booking can hold.
              </p>
            )}
          </div>
          <strong className="total">{total(showtime, seats.length)}</strong>
          <button
            className="primary"
            disabled={!seats.length}
            onClick={() => onContinue(seats)}
          >
            Continue to checkout
          </button>
        </footer>
      </main>
      {showtime.movie.posterPath && <TmdbCredit />}
    </>
  );
}
