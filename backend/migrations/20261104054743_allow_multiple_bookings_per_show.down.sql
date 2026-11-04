-- Fails if a user already has two bookings for one showing
ALTER TABLE bookings ADD CONSTRAINT bookings_event_id_user_id_key UNIQUE (event_id, user_id);
