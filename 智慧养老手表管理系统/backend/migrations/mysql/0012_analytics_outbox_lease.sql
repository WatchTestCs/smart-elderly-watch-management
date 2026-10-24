-- analytics_outbox 租约列：消费端改为"认领（短事务）→ 写 PG（无 MySQL 锁）→ 确认（短事务）"，
-- 消除 PG 写入期间持有 outbox 行锁与业务连接的窗口。触发器 INSERT 不写这两列，行为不变。
-- 崩溃后租约到期自动重投；PG 投影按 last_event_id 幂等，重投不产生重复。
ALTER TABLE analytics_outbox ADD COLUMN lease_token char(36) NULL DEFAULT NULL, ADD COLUMN lease_until datetime(6) NULL DEFAULT NULL;
ALTER TABLE analytics_outbox ADD INDEX idx_analytics_outbox_lease (lease_until);
