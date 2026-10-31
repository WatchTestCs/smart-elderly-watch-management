CREATE INDEX idx_telemetry_inbox_binding_order
    ON telemetry_inbox(device_id,binding_epoch,elder_id,status,id);
