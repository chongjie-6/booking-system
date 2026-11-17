-- Dev data: 5 films, 3 screens of 8 rows x 14 seats, and a week of showings from today (Sydney time).
-- Wipes all movies, rooms, showings and bookings first, so re-run it to move the showings forward. Never run it on production.
-- Run after the server has applied the migrations once (Git Bash):
--   docker compose exec -T postgres psql -U booking -d booking < backend/seed.sql
BEGIN;

TRUNCATE movies, rooms CASCADE; -- also empties events, seats, bookings and booking_seats

INSERT INTO rooms (name, capacity) VALUES ('Screen 1', 112), ('Screen 2', 112), ('Screen 3', 112);

INSERT INTO seats (room_id, row_label, seat_number, seat_type)
SELECT r.room_id, row_label, n,
    CASE
        WHEN row_label = 'A' AND n IN (1, 14) THEN 'wheelchair'
        WHEN row_label = 'A' AND n IN (2, 13) THEN 'companion'
        WHEN row_label = 'H' THEN 'premium'
        ELSE 'standard'
    END::seat_type
FROM rooms r, unnest(string_to_array('A,B,C,D,E,F,G,H', ',')) AS row_label, generate_series(1, 14) AS n;

-- Times are picked so showings in the same room never overlap.
WITH m (title, genre, runtime, rating, synopsis, room, times) AS (VALUES
    ('The Long Quiet', 'Drama', 128, 'M',
     'A lighthouse keeper sees out her last winter on an island the ferry no longer visits.',
     'Screen 1', ARRAY['11:00', '13:45', '16:30']),
    ('Last Train North', 'Science fiction', 141, 'M',
     'The final train out of a frozen city is carrying one passenger too many.',
     'Screen 1', ARRAY['19:00', '22:00']),
    ('Saltwater Kids', 'Animation', 88, 'G',
     'A crab who hates getting wet joins the reef swimming team.',
     'Screen 2', ARRAY['09:30', '13:30']),
    ('Kite Season', 'Comedy', 97, 'PG',
     'Two rival kite makers are forced to share one stall at the county fair.',
     'Screen 2', ARRAY['11:15', '15:30', '18:20']),
    ('The Reservoir', 'Thriller', 112, 'MA15+',
     'A salvage diver finds a sealed car at the bottom of the reservoir. Someone wants it left there.',
     'Screen 3', ARRAY['14:00', '17:45', '21:30'])
), movie AS (
    INSERT INTO movies (title, genre, runtime_minutes, rating, synopsis)
    SELECT title, genre, runtime, rating, synopsis FROM m
    RETURNING movie_id, title
)
INSERT INTO events (movie_id, room_id, price_cents, starts_at)
SELECT movie.movie_id, r.room_id, 1650,
    ((now() AT TIME ZONE 'Australia/Sydney')::date + d + t::time) AT TIME ZONE 'Australia/Sydney'
FROM m
JOIN movie USING (title)
JOIN rooms r ON r.name = m.room,
generate_series(0, 6) AS d,
unnest(m.times) AS t;

COMMIT;
