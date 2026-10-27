CREATE INDEX IF NOT EXISTS idx_refresh_tokens_user ON refresh_tokens (user_id);
-- Dropping the column also drops refresh_tokens_user_device_idx
ALTER TABLE refresh_tokens DROP COLUMN IF EXISTS device_hash;
