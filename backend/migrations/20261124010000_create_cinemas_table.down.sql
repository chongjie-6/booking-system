-- Fails if two cinemas have screens with the same name; rename them first.
ALTER TABLE rooms DROP COLUMN IF EXISTS cinema_id, ADD UNIQUE (name);
DROP TABLE IF EXISTS cinemas;
