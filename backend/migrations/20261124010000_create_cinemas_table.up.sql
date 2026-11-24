-- A site screens belong to, with what it adds to every seat's price (busy city sites charge more).
-- Format base prices are the cheapest site's, so the surcharge never goes negative and prices stay above Stripe's minimum.
CREATE TABLE IF NOT EXISTS cinemas (
    cinema_id       UUID        PRIMARY KEY DEFAULT uuidv7(),
    name            TEXT        NOT NULL UNIQUE,
    surcharge_cents INT         NOT NULL DEFAULT 0 CHECK (surcharge_cents >= 0),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER set_updated_at
BEFORE UPDATE ON cinemas
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- Existing screens go in one cinema with no surcharge, so their prices don't change.
-- Default ON DELETE (restrict) stops deleting a cinema that still has screens.
INSERT INTO cinemas (name) SELECT 'Main' WHERE EXISTS (SELECT 1 FROM rooms);
ALTER TABLE rooms ADD COLUMN cinema_id UUID REFERENCES cinemas (cinema_id);
UPDATE rooms SET cinema_id = (SELECT cinema_id FROM cinemas WHERE name = 'Main');
ALTER TABLE rooms
    ALTER COLUMN cinema_id SET NOT NULL,
    DROP CONSTRAINT rooms_name_key,
    ADD UNIQUE (cinema_id, name); -- every site has a "Screen 1"

CREATE INDEX IF NOT EXISTS rooms_cinema_idx ON rooms (cinema_id);
