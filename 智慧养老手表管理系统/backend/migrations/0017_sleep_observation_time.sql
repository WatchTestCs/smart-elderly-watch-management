-- Duration and classification can arrive independently. Unknown legacy times remain NULL.
ALTER TABLE watch_day_info ADD COLUMN IF NOT EXISTS sleep_observed_at timestamptz;
ALTER TABLE watch_day_info ADD COLUMN IF NOT EXISTS sleep_state_observed_at timestamptz;
ALTER TABLE watch_day_info_archive ADD COLUMN IF NOT EXISTS sleep_observed_at timestamptz;
ALTER TABLE watch_day_info_archive ADD COLUMN IF NOT EXISTS sleep_state_observed_at timestamptz;
