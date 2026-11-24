-- Where each cinema is. country is an ISO 3166-1 alpha-2 code (AU); lat and lng are WGS84 degrees, for maps and distance.
-- Cinemas made before this get blanks and 0, 0; fix them by hand.
ALTER TABLE cinemas
    ADD COLUMN address     TEXT             NOT NULL DEFAULT '',
    ADD COLUMN city        TEXT             NOT NULL DEFAULT '',
    ADD COLUMN state       TEXT             NOT NULL DEFAULT '',
    ADD COLUMN country     TEXT             NOT NULL DEFAULT 'AU' CHECK (country ~ '^[A-Z]{2}$'),
    ADD COLUMN postal_code TEXT             NOT NULL DEFAULT '',
    ADD COLUMN lat         DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (lat BETWEEN -90 AND 90),
    ADD COLUMN lng         DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK (lng BETWEEN -180 AND 180);

-- The defaults only filled existing rows; new cinemas must say where they are.
ALTER TABLE cinemas
    ALTER COLUMN address DROP DEFAULT, ALTER COLUMN city DROP DEFAULT, ALTER COLUMN state DROP DEFAULT,
    ALTER COLUMN country DROP DEFAULT, ALTER COLUMN postal_code DROP DEFAULT, ALTER COLUMN lat DROP DEFAULT,
    ALTER COLUMN lng DROP DEFAULT;
