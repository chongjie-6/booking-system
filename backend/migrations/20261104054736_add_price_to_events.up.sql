-- Whole cents, so totals add up without float rounding.
-- NOT NULL without a default fails if events already has rows.
ALTER TABLE events ADD COLUMN price_cents INT NOT NULL CHECK (price_cents >= 0);
