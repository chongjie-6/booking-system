-- When each cinema is open, in its local time. weekday matches Go's time.Weekday and Postgres EXTRACT(DOW): 0 = Sunday.
-- No row for a day means the cinema is closed that day. closes before opens means it closes after midnight (10:00 to 01:00).
CREATE TABLE IF NOT EXISTS cinema_hours (
    cinema_id UUID     NOT NULL REFERENCES cinemas (cinema_id) ON DELETE CASCADE,
    weekday   SMALLINT NOT NULL CHECK (weekday BETWEEN 0 AND 6),
    opens     TIME     NOT NULL,
    closes    TIME     NOT NULL CHECK (closes <> opens),
    PRIMARY KEY (cinema_id, weekday)
);
