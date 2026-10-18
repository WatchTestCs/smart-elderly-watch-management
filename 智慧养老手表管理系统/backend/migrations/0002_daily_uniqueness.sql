-- 0002: V2__daily_record_uniqueness.sql 的 PostgreSQL 翻译。
-- 归档表 + 设备每日唯一记录约束（新库无存量，直接建表与唯一索引）。

CREATE TABLE IF NOT EXISTS watch_day_info_archive (LIKE watch_day_info INCLUDING ALL);

CREATE UNIQUE INDEX IF NOT EXISTS uk_day_device_date ON watch_day_info(device_id,create_time);
