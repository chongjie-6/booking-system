-- Dev data: 15 films (details and posters filled from TMDB when TMDB_TOKEN is set), 6 screens of 8 rows x 14 seats,
-- two weeks of showings from today (Sydney time), and a few upcoming bookings for each existing user. Four films open
-- 5 to 11 days out, so the home page has some coming soon.
-- Wipes all movies, cinemas, rooms, showings and bookings first (users are kept), so re-run it to move the showings forward.
-- Never run it on production. `docker compose up` runs it once the server has applied the migrations; by hand (Git Bash):
--   docker compose exec -T postgres psql -U booking -d booking < backend/seed.sql
BEGIN;

TRUNCATE movies, cinemas CASCADE; -- also empties rooms, events, seats, bookings and booking_seats

-- A city site that charges more and a country one that doesn't. Screen names stay unique so showings below can find them.
INSERT INTO cinemas (name, surcharge_cents, address, city, state, country, postal_code, lat, lng) VALUES
    ('Melbourne Central', 400, '211 La Trobe Street', 'Melbourne', 'VIC', 'AU', '3000', -37.8102, 144.9628),
    ('Mildura', 0, '18 Langtree Avenue', 'Mildura', 'VIC', 'AU', '3500', -34.1855, 142.1610);
INSERT INTO rooms (cinema_id, name, capacity)
SELECT c.cinema_id, r.name, 112
FROM (VALUES ('Melbourne Central', 'Screen 1'), ('Melbourne Central', 'Screen 2'), ('Melbourne Central', 'Screen 3'),
    ('Mildura', 'Screen 4'), ('Mildura', 'Screen 5'), ('Mildura', 'Screen 6')) AS r (cinema, name)
JOIN cinemas c ON c.name = r.cinema;

INSERT INTO seats (room_id, row_label, seat_number, seat_type)
SELECT r.room_id, row_label, n,
    CASE
        WHEN row_label = 'A' AND n IN (1, 14) THEN 'wheelchair'
        WHEN row_label = 'A' AND n IN (2, 13) THEN 'companion'
        WHEN row_label = 'H' THEN 'premium'
        ELSE 'standard'
    END::seat_type
FROM rooms r, unnest(string_to_array('A,B,C,D,E,F,G,H', ',')) AS row_label, generate_series(1, 14) AS n;

-- Times are picked so showings in the same room never overlap. opens is the day (0 = today) of the first showing.
WITH m (title, tmdb_id, genre, runtime, rating, synopsis, room, times, opens) AS (VALUES
    ('Parasite', 496243, 'Thriller', 133, 'MA15+',
     'A family scraping by in a basement flat talks its way, one job at a time, into a rich household.',
     'Screen 1', ARRAY['11:00', '13:45', '16:30'], 0),
    ('Interstellar', 157336, 'Science fiction', 169, 'M',
     'With crops failing on Earth, a former pilot leads a crew through a wormhole to look for a new home.',
     'Screen 1', ARRAY['19:00', '22:00'], 0),
    ('Toy Story', 862, 'Animation', 81, 'G',
     'A pull-string cowboy and a new space ranger toy compete to be their boy''s favourite.',
     'Screen 2', ARRAY['09:30', '13:30'], 0),
    ('The Grand Budapest Hotel', 120467, 'Comedy', 100, 'M',
     'A celebrated concierge and his lobby boy are drawn into a theft and a fight over a family fortune.',
     'Screen 2', ARRAY['11:15', '15:30', '18:20'], 0),
    ('Spider-Man: Into the Spider-Verse', 324857, 'Animation', 117, 'PG',
     'A Brooklyn teenager gets spider powers and meets other Spider-People pulled in from other dimensions.',
     'Screen 2', ARRAY['20:30'], 0),
    ('Spirited Away', 129, 'Animation', 125, 'PG',
     'A girl wanders into a spirit world and takes a job in a bathhouse to free her parents.',
     'Screen 3', ARRAY['11:00'], 0),
    ('The Dark Knight', 155, 'Action', 152, 'M',
     'Batman, a police lieutenant and a new district attorney take on a criminal who wants Gotham in chaos.',
     'Screen 3', ARRAY['14:00', '17:45', '21:30'], 0),
    ('Paddington 2', 346648, 'Family', 104, 'PG',
     'Paddington takes odd jobs to buy his aunt a pop-up book, then is jailed when it is stolen.',
     'Screen 4', ARRAY['10:00', '15:15'], 0),
    ('Dune: Part Two', 693134, 'Science fiction', 166, 'M',
     'Paul Atreides joins the Fremen of Arrakis and seeks revenge on the house that destroyed his family.',
     'Screen 4', ARRAY['12:00', '18:30'], 0),
    ('Inception', 27205, 'Science fiction', 148, 'M',
     'A thief who steals secrets from people''s dreams is hired to plant an idea instead.',
     'Screen 5', ARRAY['10:00', '16:00'], 0),
    ('Everything Everywhere All at Once', 545611, 'Comedy', 139, 'MA15+',
     'A laundromat owner in the middle of a tax audit finds she can borrow skills from her other lives across the multiverse.',
     'Screen 5', ARRAY['13:00', '20:15'], 0),
    ('Coco', 354912, 'Animation', 105, 'PG',
     'A boy who dreams of playing music follows his great-great-grandfather''s guitar into the Land of the Dead.',
     'Screen 6', ARRAY['10:00', '14:00'], 5),
    ('La La Land', 313369, 'Musical', 128, 'PG',
     'A jazz pianist and an aspiring actor fall in love in Los Angeles while chasing their dreams.',
     'Screen 6', ARRAY['11:50'], 7),
    ('Arrival', 329865, 'Science fiction', 116, 'M',
     'A linguist is brought in to talk with the visitors after twelve alien craft touch down around the world.',
     'Screen 6', ARRAY['16:00'], 9),
    ('Oppenheimer', 872585, 'Drama', 180, 'M',
     'The physicist J. Robert Oppenheimer leads the race to build the atomic bomb, then lives with what it did.',
     'Screen 6', ARRAY['18:30'], 11)
), movie AS (
    INSERT INTO movies (title, tmdb_id, genre, runtime_minutes, rating, synopsis)
    SELECT title, tmdb_id, genre, runtime, rating, synopsis FROM m
    RETURNING movie_id, title
)
INSERT INTO events (movie_id, room_id, format_id, starts_at)
SELECT movie.movie_id, r.room_id, (SELECT format_id FROM event_formats WHERE name = 'Standard'),
    ((now() AT TIME ZONE 'Australia/Sydney')::date + d + t::time) AT TIME ZONE 'Australia/Sydney'
FROM m
JOIN movie USING (title)
JOIN rooms r ON r.name = m.room,
generate_series(m.opens, 13) AS d,
unnest(m.times) AS t;

UPDATE movies SET featured = true WHERE title IN ('Interstellar', 'Dune: Part Two');

-- A few films in other formats, so prices vary. The formats and their prices come from the migrations.
UPDATE events e SET format_id = f.format_id
FROM movies m, event_formats f, (VALUES
    ('Interstellar', 'IMAX'), ('Dune: Part Two', 'IMAX'), ('Spider-Man: Into the Spider-Verse', '3D'), ('The Dark Knight', '4D')
) AS x (title, format)
WHERE m.movie_id = e.movie_id AND m.title = x.title AND f.name = x.format;

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
INSERT INTO booking_seats (booking_id, event_id, room_id, seat_id, price_cents)
SELECT b.booking_id, b.event_id, e.room_id, s.seat_id, f.base_price_cents + sts.surcharge_cents
FROM b
JOIN u USING (user_id)
JOIN events e USING (event_id)
JOIN event_formats f USING (format_id)
JOIN seats s ON s.room_id = e.room_id AND s.row_label = chr(65 + u.n::int) AND s.seat_number IN (7, 8)
JOIN seat_type_surcharges sts USING (seat_type);

COMMIT;
