ALTER TABLE movies
    DROP COLUMN IF EXISTS featured,
    DROP COLUMN IF EXISTS trailer_key,
    DROP COLUMN IF EXISTS backdrop_path;
