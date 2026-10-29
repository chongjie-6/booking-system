CREATE TYPE seat_type AS ENUM ('standard', 'wheelchair', 'companion', 'premium');

CREATE TABLE IF NOT EXISTS seats (
    seat_id UUID PRIMARY KEY DEFAULT uuidv7(),
    room_id UUID NOT NULL REFERENCES rooms(room_id) ON DELETE CASCADE,
    row_label TEXT NOT NULL,
    seat_number INT NOT NULL CHECK (seat_number > 0),
    seat_type seat_type NOT NULL DEFAULT 'standard',
    UNIQUE (room_id, row_label, seat_number),
    UNIQUE (seat_id, room_id) -- target for booking_seats (seat_id, room_id) FK
);