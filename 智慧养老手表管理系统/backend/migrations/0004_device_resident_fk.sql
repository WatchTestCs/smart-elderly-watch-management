-- Reject orphan device bindings. Existing invalid rows must be repaired explicitly;
-- this migration never deletes devices or silently clears their ownership.
ALTER TABLE device ADD CONSTRAINT fk_device_resident
  FOREIGN KEY (resident_id) REFERENCES resident(resident_id) ON DELETE RESTRICT;
