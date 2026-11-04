-- Only the SHA-256 of each token is stored, like refresh tokens. Using a token deletes it, so a link works once.
CREATE TABLE IF NOT EXISTS password_reset_tokens (
    token_hash CHAR(64)    PRIMARY KEY,
    user_id    UUID        NOT NULL REFERENCES users (user_id) ON DELETE CASCADE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Postgres doesn't index foreign keys on its own; a reset deletes the user's other tokens by user_id
CREATE INDEX IF NOT EXISTS password_reset_tokens_user_idx ON password_reset_tokens (user_id);
