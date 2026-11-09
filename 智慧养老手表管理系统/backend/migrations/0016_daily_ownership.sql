-- Freeze the currently visible legacy association. Earlier ownership cannot be reconstructed.
ALTER TABLE watch_day_info ADD COLUMN IF NOT EXISTS old_id bigint;
ALTER TABLE watch_day_info ADD COLUMN IF NOT EXISTS binding_epoch bigint NOT NULL DEFAULT 0;
ALTER TABLE watch_day_info_archive ADD COLUMN IF NOT EXISTS step_observed_at timestamptz;
ALTER TABLE watch_day_info_archive ADD COLUMN IF NOT EXISTS old_id bigint;
ALTER TABLE watch_day_info_archive ADD COLUMN IF NOT EXISTS binding_epoch bigint NOT NULL DEFAULT 0;
UPDATE watch_day_info w SET old_id=d.old_id,binding_epoch=d.health_epoch
FROM device d WHERE d.device_id=w.device_id AND w.old_id IS NULL AND w.binding_epoch=0 AND d.old_id IS NOT NULL;
-- Mark unassigned legacy rows so replay cannot assign them after a later binding.
UPDATE watch_day_info SET binding_epoch=-1 WHERE old_id IS NULL AND binding_epoch=0;
SELECT setval(pg_get_serial_sequence('watch_day_info_archive','id'),COALESCE(MAX(id),0)+1,false) FROM watch_day_info_archive;
CREATE INDEX IF NOT EXISTS idx_day_owner ON watch_day_info(old_id,device_id,create_time);
CREATE INDEX IF NOT EXISTS idx_day_archive_owner ON watch_day_info_archive(old_id,device_id,create_time);
