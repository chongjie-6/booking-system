DROP INDEX IF EXISTS outbox_created_at_idx;

-- Existing rows count as sent, so going back to the polling relay doesn't resend them.
ALTER TABLE outbox ADD COLUMN published_at TIMESTAMPTZ;
UPDATE outbox SET published_at = created_at;
CREATE INDEX outbox_unpublished_idx ON outbox (outbox_id) WHERE published_at IS NULL;
