-- Holidays and one-off hours: a row replaces cinema_hours for that cinema on that date, in its local time.
-- opens and closes both NULL means closed all day. note is shown to customers, e.g. 'Christmas Day'.
CREATE TABLE IF NOT EXISTS cinema_hour_exceptions (
    cinema_id UUID NOT NULL REFERENCES cinemas (cinema_id) ON DELETE CASCADE,
    day       DATE NOT NULL,
    opens     TIME,
    closes    TIME,
    note      TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (cinema_id, day),
    CHECK ((opens IS NULL) = (closes IS NULL)),
    CHECK (closes <> opens)
);
