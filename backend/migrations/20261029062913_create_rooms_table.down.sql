-- Copy each room's capacity back onto its events before dropping rooms
ALTER TABLE events ADD COLUMN capacity INT;
UPDATE events e SET capacity = r.capacity FROM rooms r WHERE r.room_id = e.room_id;
ALTER TABLE events
    ALTER COLUMN capacity SET NOT NULL,
    ADD CHECK (capacity >= 0);

-- Dropping the column also drops events_room_idx, the foreign key and the (event_id, room_id) unique
ALTER TABLE events DROP COLUMN room_id;
DROP TABLE IF EXISTS rooms;
