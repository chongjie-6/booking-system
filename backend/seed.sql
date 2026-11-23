-- Dev data: 11 films (details and posters filled from TMDB when TMDB_TOKEN is set), 5 screens of 8 rows x 14 seats,
-- two weeks of showings from today (Sydney time), and a few upcoming bookings for each existing user.
-- Wipes all movies, rooms, showings and bookings first (users are kept), so re-run it to move the showings forward.
-- Never run it on production. `docker compose up` runs it once the server has applied the migrations; by hand (Git Bash):
--   docker compose exec -T postgres psql -U booking -d booking < backend/seed.sql
BEGIN;

TRUNCATE movies, rooms CASCADE; -- also empties events, seats, bookings and booking_seats

INSERT INTO rooms (name, capacity) VALUES
    ('Screen 1', 112), ('Screen 2', 112), ('Screen 3', 112), ('Screen 4', 112), ('Screen 5', 112);

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
    ('Spider-Man: Into the Spider-Verse', 324857, 'Animation', 117, 'PG',
     'A Brooklyn teenager gets spider powers and meets other Spider-People pulled in from other dimensions.',
     'Screen 2', ARRAY['20:30']),
    ('Spirited Away', 129, 'Animation', 125, 'PG',
     'A girl wanders into a spirit world and takes a job in a bathhouse to free her parents.',
     'Screen 3', ARRAY['11:00']),
    ('The Dark Knight', 155, 'Action', 152, 'M',
     'Batman, a police lieutenant and a new district attorney take on a criminal who wants Gotham in chaos.',
     'Screen 3', ARRAY['14:00', '17:45', '21:30']),
    ('Paddington 2', 346648, 'Family', 104, 'PG',
     'Paddington takes odd jobs to buy his aunt a pop-up book, then is jailed when it is stolen.',
     'Screen 4', ARRAY['10:00', '15:15']),
    ('Dune: Part Two', 693134, 'Science fiction', 166, 'M',
     'Paul Atreides joins the Fremen of Arrakis and seeks revenge on the house that destroyed his family.',
     'Screen 4', ARRAY['12:00', '18:30']),
    ('Inception', 27205, 'Science fiction', 148, 'M',
     'A thief who steals secrets from people''s dreams is hired to plant an idea instead.',
     'Screen 5', ARRAY['10:00', '16:00']),
    ('Everything Everywhere All at Once', 545611, 'Comedy', 139, 'MA15+',
     'A laundromat owner in the middle of a tax audit finds she can borrow skills from her other lives across the multiverse.',
     'Screen 5', ARRAY['13:00', '20:15'])
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
generate_series(0, 13) AS d,
unnest(m.times) AS t;

UPDATE movies SET featured = true WHERE title IN ('Interstellar', 'Dune: Part Two');

-- Each existing user gets the same five upcoming bookings of two seats, unpaid (no PaymentIntent, so cancelling skips
-- the refund). User n (oldest first) sits in row B, C, ... so their seats never clash.
-- ponytail: only the 7 oldest users get bookings, one per row B-H; pick seats per user if more are needed.
WITH u AS (
    SELECT user_id, row_number() OVER (ORDER BY created_at) AS n FROM users
), pick (title, d, t) AS (VALUES
    ('Interstellar', 1, '19:00'),
    ('Dune: Part Two', 2, '18:30'),
    ('Spirited Away', 4, '11:00'),
    ('The Dark Knight', 6, '21:30'),
    ('Everything Everywhere All at Once', 9, '20:15')
), b AS (
    INSERT INTO bookings (event_id, user_id)
    SELECT e.event_id, u.user_id
    FROM u
    CROSS JOIN pick
    JOIN movies mv ON mv.title = pick.title
    JOIN events e ON e.movie_id = mv.movie_id
        AND e.starts_at = ((now() AT TIME ZONE 'Australia/Sydney')::date + pick.d + pick.t::time) AT TIME ZONE 'Australia/Sydney'
    WHERE u.n <= 7
    RETURNING booking_id, event_id, user_id
)
INSERT INTO booking_seats (booking_id, event_id, room_id, seat_id)
SELECT b.booking_id, b.event_id, e.room_id, s.seat_id
FROM b
JOIN u USING (user_id)
JOIN events e USING (event_id)
JOIN seats s ON s.room_id = e.room_id AND s.row_label = chr(65 + u.n::int) AND s.seat_number IN (7, 8);

COMMIT;
