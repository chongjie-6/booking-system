-- The short ref (last 8 hex digits of booking_id, see booking.Create) must be unique within a showing, so it can be
-- looked up at the door alongside the event ID
CREATE UNIQUE INDEX IF NOT EXISTS bookings_event_ref_idx ON bookings (event_id, upper(right(booking_id::text, 8)));
