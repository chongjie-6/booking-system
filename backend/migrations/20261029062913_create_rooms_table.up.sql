CREATE TABLE IF NOT EXISTS rooms (
    room_id    UUID        PRIMARY KEY DEFAULT uuidv7(),
    name       TEXT        NOT NULL UNIQUE,
    capacity   INT         NOT NULL CHECK (capacity > 0),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER set_updated_at
BEFORE UPDATE ON rooms
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Capacity now lives on the room; seats_left stays per showing.
-- NOT NULL without a default fails if events already has rows.
-- Default ON DELETE (restrict) stops deleting a room that still has showings.
ALTER TABLE events
    ADD COLUMN room_id UUID NOT NULL REFERENCES rooms (room_id),
    ADD UNIQUE (event_id, room_id), -- target for booking_seats (event_id, room_id) FK
    DROP COLUMN capacity;

CREATE INDEX IF NOT EXISTS events_room_idx ON events (room_id);
