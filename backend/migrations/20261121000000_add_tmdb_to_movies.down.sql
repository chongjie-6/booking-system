ALTER TABLE movies
    DROP COLUMN IF EXISTS tmdb_id,
    DROP COLUMN IF EXISTS poster_path;
