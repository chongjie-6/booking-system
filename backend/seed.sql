-- Dev data: 5 films (details and posters filled from TMDB when TMDB_TOKEN is set), 3 screens of 8 rows x 14 seats, and a week of showings from today (Sydney time).
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
WITH m (title, tmdb_id, genre, runtime, rating, synopsis, room, times) AS (VALUES
    ('Parasite', 496243, 'Thriller', 133, 'MA15+',
     'A family scraping by in a basement flat talks its way, one job at a time, into a rich household.',
     'Screen 1', ARRAY['11:00', '13:45', '16:30']),
    ('Interstellar', 157336, 'Science fiction', 169, 'M',
     'With crops failing on Earth, a former pilot leads a crew through a wormhole to look for a new home.',
     'Screen 1', ARRAY['19:00', '22:00']),
    ('Toy Story', 862, 'Animation', 81, 'G',
     'A pull-string cowboy and a new space ranger toy compete to be their boy''s favourite.',
     'Screen 2', ARRAY['09:30', '13:30']),
    ('The Grand Budapest Hotel', 120467, 'Comedy', 100, 'M',
     'A celebrated concierge and his lobby boy are drawn into a theft and a fight over a family fortune.',
     'Screen 2', ARRAY['11:15', '15:30', '18:20']),
    ('The Dark Knight', 155, 'Action', 152, 'M',
     'Batman, a police lieutenant and a new district attorney take on a criminal who wants Gotham in chaos.',
     'Screen 3', ARRAY['14:00', '17:45', '21:30'])
), movie AS (
    INSERT INTO movies (title, tmdb_id, genre, runtime_minutes, rating, synopsis)
    SELECT title, tmdb_id, genre, runtime, rating, synopsis FROM m
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

UPDATE movies SET featured = true WHERE title = 'Interstellar';

COMMIT;
