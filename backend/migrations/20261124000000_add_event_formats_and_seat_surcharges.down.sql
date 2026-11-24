ALTER TABLE events ADD COLUMN price_cents INT CHECK (price_cents >= 0);
UPDATE events e SET price_cents = f.base_price_cents FROM event_formats f WHERE f.format_id = e.format_id;
ALTER TABLE events ALTER COLUMN price_cents SET NOT NULL, DROP COLUMN format_id;

ALTER TABLE booking_seats DROP COLUMN IF EXISTS price_cents;
ALTER TABLE seats DROP CONSTRAINT IF EXISTS seats_seat_type_fkey;
DROP TABLE IF EXISTS seat_type_surcharges;
DROP TABLE IF EXISTS event_formats;
