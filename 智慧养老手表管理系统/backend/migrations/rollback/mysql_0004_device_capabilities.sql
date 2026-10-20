-- Offline only: stop API/workers, back up these tables and care incident/event data first.
-- MySQL DDL implicitly commits. This is compensating removal, not transactional undo.
-- Device actions already executed cannot be rolled back. Retain generated care incidents.
DROP TABLE IF EXISTS alert_review;
DROP TABLE IF EXISTS fence_event;
DROP TABLE IF EXISTS fence_state;
DROP TABLE IF EXISTS device_fence;
DROP TABLE IF EXISTS metric_daily;
DROP TABLE IF EXISTS device_command;
DROP TABLE IF EXISTS device_command_profile;
DELETE FROM schema_migrations WHERE name='mysql/0004_device_capabilities.sql';
