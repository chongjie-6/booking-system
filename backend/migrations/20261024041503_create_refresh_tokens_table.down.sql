-- Dropping the table also drops its indexes and trigger. Keep update_updated_at_column(), users still uses it
DROP TABLE IF EXISTS refresh_tokens;
