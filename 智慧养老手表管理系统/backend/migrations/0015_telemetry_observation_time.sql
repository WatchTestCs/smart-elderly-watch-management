ALTER TABLE watch_day_info ADD COLUMN IF NOT EXISTS step_observed_at timestamptz;
CREATE INDEX IF NOT EXISTS idx_loc_resident_observed ON device_location(resident_id,event_time,id);
CREATE INDEX IF NOT EXISTS idx_loc_device_observed ON device_location(device_id,event_time,id);
