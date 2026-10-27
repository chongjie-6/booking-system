ALTER TABLE refresh_tokens ADD COLUMN IF NOT EXISTS device_hash CHAR(64);

-- Leads with user_id, so it also serves the user_id lookups idx_refresh_tokens_user did
CREATE INDEX IF NOT EXISTS refresh_tokens_user_device_idx ON refresh_tokens (user_id, device_hash);
DROP INDEX IF EXISTS idx_refresh_tokens_user;
