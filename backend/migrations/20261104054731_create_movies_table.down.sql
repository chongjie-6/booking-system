-- Copy each showing's title back from its movie before dropping the link
ALTER TABLE events ADD COLUMN title TEXT;
UPDATE events e SET title = m.title FROM movies m WHERE m.movie_id = e.movie_id;
ALTER TABLE events ALTER COLUMN title SET NOT NULL;

ALTER TABLE events DROP COLUMN movie_id; -- also drops events_movie_idx
DROP TABLE IF EXISTS movies;
