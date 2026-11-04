-- A user can book the same showing more than once; booking_seats still stops a seat being sold twice.
-- bookings_user_idx still serves "my bookings" lookups.
ALTER TABLE bookings DROP CONSTRAINT bookings_event_id_user_id_key;
