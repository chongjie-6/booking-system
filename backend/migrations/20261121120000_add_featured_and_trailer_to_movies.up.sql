-- trailer_key (a YouTube video key) and backdrop_path are NULL until resource.FillFromTMDB has fetched the film, and '' if TMDB has none
ALTER TABLE movies
    ADD COLUMN IF NOT EXISTS featured      BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN IF NOT EXISTS trailer_key   TEXT,
    ADD COLUMN IF NOT EXISTS backdrop_path TEXT;
