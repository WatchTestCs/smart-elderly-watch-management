ALTER TABLE watch_day_info ADD COLUMN IF NOT EXISTS step_observed_at timestamptz;
CREATE INDEX IF NOT EXISTS idx_loc_old_observed ON device_location(old_id,be_time,id);
CREATE INDEX IF NOT EXISTS idx_loc_device_observed ON device_location(device_id,be_time,id);
