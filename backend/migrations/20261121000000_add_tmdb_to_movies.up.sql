-- poster_path is NULL until resource.FillFromTMDB has fetched the film, and '' if TMDB has no poster for it
ALTER TABLE movies
    ADD COLUMN IF NOT EXISTS tmdb_id     INT UNIQUE,
    ADD COLUMN IF NOT EXISTS poster_path TEXT;
