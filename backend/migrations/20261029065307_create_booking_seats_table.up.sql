ALTER TABLE bookings ADD CONSTRAINT bookings_booking_event_key UNIQUE (booking_id, event_id);

-- Composite FKs pin each seat to its booking's event and to that event's room
CREATE TABLE IF NOT EXISTS booking_seats (
    booking_id UUID NOT NULL,
    event_id   UUID NOT NULL,
    room_id    UUID NOT NULL,
    seat_id    UUID NOT NULL,
    PRIMARY KEY (booking_id, seat_id),
    UNIQUE (event_id, seat_id),
    FOREIGN KEY (booking_id, event_id) REFERENCES bookings (booking_id, event_id) ON DELETE CASCADE,
    FOREIGN KEY (event_id, room_id)    REFERENCES events (event_id, room_id) ON DELETE CASCADE,
    FOREIGN KEY (seat_id, room_id)     REFERENCES seats (seat_id, room_id)
);
