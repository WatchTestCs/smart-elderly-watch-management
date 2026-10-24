-- 与 mysql/0010_care_recipient_key.sql 对齐（PostgreSQL 供遗留迁移与回归工具使用）。
-- 回填摘要用 bytea 拼接 0x00 分隔符：PostgreSQL 文本不允许 NUL 字节，
-- 而 securestore 无密钥回退的摘要布局是 sha256(scope || 0x00 || value)。
ALTER TABLE care_notification_outbox ADD COLUMN recipient_key char(64) NOT NULL DEFAULT '';
UPDATE care_notification_outbox SET recipient_key=encode(sha256(convert_to('care-notification-recipient','UTF8') || '\x00'::bytea || convert_to(recipient,'UTF8')),'hex');
ALTER TABLE care_notification_outbox DROP CONSTRAINT care_notification_outbox_incident_id_channel_recipient_key;
ALTER TABLE care_notification_outbox ADD CONSTRAINT care_notification_outbox_recipient_key_uniq UNIQUE (incident_id, channel, recipient_key);
