-- The kind of showing (Standard, 3D, IMAX, 4D) and what a seat at one costs before its seat type's surcharge.
-- Stripe's smallest charge in AUD is 50 cents. Staff manage these on the admin page.
CREATE TABLE IF NOT EXISTS event_formats (
    format_id        UUID PRIMARY KEY DEFAULT uuidv7(),
    name             TEXT NOT NULL UNIQUE,
    base_price_cents INT  NOT NULL CHECK (base_price_cents >= 50)
);
INSERT INTO event_formats (name, base_price_cents) VALUES
    ('Standard', 1650), ('3D', 1950), ('IMAX', 2250), ('4D', 2850);

-- What each seat type adds to the format's price. The foreign key from seats means every type in use has a row,
-- so a new seat_type needs one here before seats can use it.
CREATE TABLE IF NOT EXISTS seat_type_surcharges (
    seat_type       seat_type PRIMARY KEY,
    surcharge_cents INT NOT NULL CHECK (surcharge_cents >= 0)
);
INSERT INTO seat_type_surcharges (seat_type, surcharge_cents) VALUES
    ('standard', 0), ('wheelchair', 0), ('companion', 0), ('premium', 300);
ALTER TABLE seats ADD FOREIGN KEY (seat_type) REFERENCES seat_type_surcharges (seat_type);

-- What each seat sold for, so changing a price later doesn't change past bookings, their emails or their refunds.
ALTER TABLE booking_seats ADD COLUMN price_cents INT CHECK (price_cents >= 0);
UPDATE booking_seats bs SET price_cents = e.price_cents FROM events e WHERE e.event_id = bs.event_id;
ALTER TABLE booking_seats ALTER COLUMN price_cents SET NOT NULL;

-- A showing's price now comes from its format. Existing showings become Standard and take its price.
ALTER TABLE events ADD COLUMN format_id UUID REFERENCES event_formats (format_id);
UPDATE events SET format_id = (SELECT format_id FROM event_formats WHERE name = 'Standard');
ALTER TABLE events ALTER COLUMN format_id SET NOT NULL, DROP COLUMN price_cents;
