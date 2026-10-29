CREATE TABLE IF NOT EXISTS events(
    event_id UUID PRIMARY KEY DEFAULT uuidv7(),
    title TEXT NOT NULL,
    capacity INT NOT NULL CHECK(capacity >= 0),
    seats_left INT NOT NULL CHECK(seats_left >= 0),
    starts_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER set_updated_at
BEFORE UPDATE ON events
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();