-- Nothing kept seats_left in step with bookings. Seats left is the room's seat count minus the showing's booking_seats rows.
ALTER TABLE events DROP COLUMN IF EXISTS seats_left;
