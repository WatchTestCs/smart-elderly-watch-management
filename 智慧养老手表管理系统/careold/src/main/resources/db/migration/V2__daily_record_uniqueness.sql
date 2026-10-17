-- Preserve older duplicate records before enforcing one device record per day.
CREATE TABLE watch_day_info_archive AS SELECT * FROM watch_day_info WHERE 1=0;
INSERT INTO watch_day_info_archive
SELECT d.* FROM watch_day_info d
WHERE d.device_id IS NOT NULL AND d.create_time IS NOT NULL
AND EXISTS (SELECT 1 FROM watch_day_info newer
    WHERE newer.device_id=d.device_id AND newer.create_time=d.create_time AND newer.id>d.id);
DELETE FROM watch_day_info WHERE id IN (SELECT id FROM watch_day_info_archive);
CREATE UNIQUE INDEX IF NOT EXISTS uk_day_device_date ON watch_day_info(device_id,create_time);
UPDATE app_metadata SET schema_version=2 WHERE id=1;
