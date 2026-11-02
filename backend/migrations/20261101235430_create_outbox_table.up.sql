CREATE TABLE IF NOT EXISTS outbox (
    outbox_id UUID PRIMARY KEY DEFAULT uuidv7(),
    topic TEXT NOT NULL,
    key TEXT NOT NULL,
    payload JSONB NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ
);

CREATE INDEX outbox_unpublished_idx ON outbox (outbox_id)
WHERE
    published_at IS NULL;