-- The Stripe PaymentIntent that paid for the booking. UNIQUE makes booking idempotent: the checkout page and Stripe's
-- webhook both confirm the same payment, and whichever comes second finds the booking already made.
-- NULL for bookings made before payments.
ALTER TABLE bookings ADD COLUMN IF NOT EXISTS payment_intent_id TEXT UNIQUE;
