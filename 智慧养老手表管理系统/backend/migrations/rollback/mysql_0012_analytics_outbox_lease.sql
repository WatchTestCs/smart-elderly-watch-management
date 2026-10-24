-- 回滚租约改造：确认 outbox 无未确认的租约后执行（存在租约行时先 SET NULL 释放）。
ALTER TABLE analytics_outbox DROP INDEX idx_analytics_outbox_lease;
ALTER TABLE analytics_outbox DROP COLUMN lease_token, DROP COLUMN lease_until;
DELETE FROM schema_migrations WHERE name='mysql/0012_analytics_outbox_lease.sql';
