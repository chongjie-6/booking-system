-- Dropping the table also drops its index and trigger
DROP TABLE IF EXISTS users;
DROP FUNCTION IF EXISTS update_updated_at_column();
