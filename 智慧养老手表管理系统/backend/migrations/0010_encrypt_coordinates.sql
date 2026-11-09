-- 与 mysql/0011_encrypt_coordinates.sql 对齐（PostgreSQL 供遗留迁移与回归工具使用）。
ALTER TABLE device_location ALTER COLUMN lng TYPE varchar(128), ALTER COLUMN lat TYPE varchar(128);
ALTER TABLE device ALTER COLUMN last_lng TYPE varchar(128), ALTER COLUMN last_lat TYPE varchar(128);
ALTER TABLE alarm_precess ALTER COLUMN lng TYPE varchar(128), ALTER COLUMN lat TYPE varchar(128);
ALTER TABLE alarm_process ALTER COLUMN lng TYPE varchar(128), ALTER COLUMN lat TYPE varchar(128);
