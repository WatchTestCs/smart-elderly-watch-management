-- Additive: keep retired device epochs separate from the current alert view.
ALTER TABLE ratepre ADD COLUMN binding_epoch bigint NOT NULL DEFAULT 1;
-- Only observations within the currently recorded binding may be promoted
-- into the active alert epoch. Pre-binding history remains historical.
UPDATE ratepre r INNER JOIN device d ON d.device_id=r.device_id
SET r.binding_epoch=d.health_epoch
WHERE r.old_id=d.old_id AND d.bind_time IS NOT NULL AND r.create_time>=d.bind_time;
CREATE INDEX idx_ratepre_binding_current ON ratepre(device_id,binding_epoch,old_id,create_time);
-- Atomic acknowledgements for all numbered reports, including vitals and falls.
CREATE TABLE IF NOT EXISTS device_report_receipt (
  device_id bigint NOT NULL,
  binding_epoch bigint NOT NULL,
  request_id varchar(64) CHARACTER SET utf8mb4 COLLATE utf8mb4_bin NOT NULL,
  payload_digest char(64) NOT NULL,
  result_json longtext NOT NULL,
  created_at datetime(6) NOT NULL DEFAULT CURRENT_TIMESTAMP(6),
  PRIMARY KEY(device_id,binding_epoch,request_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
CREATE INDEX idx_telemetry_inbox_device_order ON telemetry_inbox(device_id,status,id);
