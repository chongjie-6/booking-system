CREATE TABLE IF NOT EXISTS movies (
    movie_id        UUID        PRIMARY KEY DEFAULT uuidv7(),
    title           TEXT        NOT NULL,
    genre           TEXT        NOT NULL,
    runtime_minutes INT         NOT NULL CHECK (runtime_minutes > 0),
    rating          TEXT        NOT NULL, -- classification, e.g. PG or M
    synopsis        TEXT        NOT NULL DEFAULT '',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE OR REPLACE TRIGGER set_updated_at
BEFORE UPDATE ON movies
FOR EACH ROW
EXECUTE FUNCTION update_updated_at_column();

-- A showing's title now comes from its movie.
-- NOT NULL without a default fails if events already has rows.
-- Default ON DELETE (restrict) stops deleting a movie that still has showings.
ALTER TABLE events
    ADD COLUMN movie_id UUID NOT NULL REFERENCES movies (movie_id),
    DROP COLUMN title;

CREATE INDEX IF NOT EXISTS events_movie_idx ON events (movie_id);
