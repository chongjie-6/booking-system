ALTER TABLE events ADD COLUMN seats_left INT NOT NULL DEFAULT 0 CHECK (seats_left >= 0);
UPDATE events e SET seats_left =
    (SELECT count(*) FROM seats s WHERE s.room_id = e.room_id) -
    (SELECT count(*) FROM booking_seats bs WHERE bs.event_id = e.event_id);
ALTER TABLE events ALTER COLUMN seats_left DROP DEFAULT;
