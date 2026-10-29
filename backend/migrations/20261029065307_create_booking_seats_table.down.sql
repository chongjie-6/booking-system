DROP TABLE IF EXISTS booking_seats;
ALTER TABLE bookings DROP CONSTRAINT IF EXISTS bookings_booking_event_key;
