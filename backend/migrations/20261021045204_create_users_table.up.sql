CREATE TABLE IF NOT EXISTS users (
    user_id         UUID        PRIMARY KEY DEFAULT uuidv7(),
    user_role       TEXT        NOT NULL DEFAULT 'user',
    hashed_password TEXT,                -- nullable, matches *string in Go
    first_name      TEXT        NOT NULL,
    last_name       TEXT        NOT NULL,
    email           TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Case-insensitive unique email, so Bob@x.com and bob@x.com can't both sign up
CREATE UNIQUE INDEX IF NOT EXISTS users_email_key ON users (lower(email));

-- Add trigger automatically updating the updated_at column
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN 
    NEW.updated_at = now();
    RETURN NEW;
END
$$ LANGUAGE plpgsql;

-- Add trigger to table
CREATE OR REPLACE TRIGGER set_updated_at 
BEFORE UPDATE ON users 
FOR EACH ROW 
EXECUTE FUNCTION update_updated_at_column();
