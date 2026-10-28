CREATE TABLE IF NOT EXISTS telemetry_ingest_queue (
    id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    device_id BIGINT NOT NULL,
    old_id BIGINT NOT NULL,
    payload LONGTEXT NOT NULL,
    state VARCHAR(16) NOT NULL DEFAULT 'pending',
    attempts INT NOT NULL DEFAULT 0,
    next_attempt_at DATETIME(6) NOT NULL,
    created_at DATETIME(6) NOT NULL,
    last_error VARCHAR(64) NOT NULL DEFAULT '',
    INDEX idx_telemetry_ingest_due (state,next_attempt_at,id),
    INDEX idx_telemetry_ingest_device_order (device_id,id)
) ENGINE=InnoDB;
