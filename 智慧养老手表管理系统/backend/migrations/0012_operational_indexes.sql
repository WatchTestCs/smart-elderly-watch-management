CREATE INDEX IF NOT EXISTS idx_care_outbox_age ON care_notification_outbox(status,created_at);
