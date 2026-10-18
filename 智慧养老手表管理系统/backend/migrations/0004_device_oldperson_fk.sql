-- Reject orphan device bindings. Existing invalid rows must be repaired explicitly;
-- this migration never deletes devices or silently clears their ownership.
ALTER TABLE device ADD CONSTRAINT fk_device_oldperson
  FOREIGN KEY (old_id) REFERENCES oldperson(old_id) ON DELETE RESTRICT;
