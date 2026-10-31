-- PostgreSQL integration-compatible representation; MySQL remains primary.
ALTER TABLE ratepre ADD COLUMN binding_epoch bigint NOT NULL DEFAULT 1;
-- Historical association before the most recent bind is not reconstructible.
UPDATE ratepre r SET binding_epoch=d.health_epoch FROM device d
WHERE d.device_id=r.device_id AND r.old_id=d.old_id
  AND d.bind_time IS NOT NULL AND r.create_time>=d.bind_time;
CREATE INDEX idx_ratepre_binding_current ON ratepre(device_id,binding_epoch,old_id,create_time);
CREATE TABLE IF NOT EXISTS device_report_receipt (
  device_id bigint NOT NULL,
  binding_epoch bigint NOT NULL,
  request_id varchar(64) NOT NULL,
  payload_digest char(64) NOT NULL,
  result_json text NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now(),
  PRIMARY KEY(device_id,binding_epoch,request_id)
);
CREATE INDEX idx_telemetry_inbox_device_order ON telemetry_inbox(device_id,status,id);
