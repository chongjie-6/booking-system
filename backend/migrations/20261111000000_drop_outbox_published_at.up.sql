-- Debezium reads new outbox rows from the WAL, so rows are no longer marked as sent.
DROP INDEX IF EXISTS outbox_unpublished_idx;
ALTER TABLE outbox DROP COLUMN IF EXISTS published_at;

-- For the hourly prune, which deletes by age.
CREATE INDEX outbox_created_at_idx ON outbox (created_at);
