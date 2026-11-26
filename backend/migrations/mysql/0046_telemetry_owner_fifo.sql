-- Owner-specific pending queue prefix and ordered locking reads.
ALTER TABLE telemetry_inbox ADD INDEX idx_telemetry_inbox_owner_pending (elder_id,status,id);
