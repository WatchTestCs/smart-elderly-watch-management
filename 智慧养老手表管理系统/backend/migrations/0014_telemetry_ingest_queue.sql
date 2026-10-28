CREATE TABLE IF NOT EXISTS telemetry_ingest_queue (
    id BIGSERIAL PRIMARY KEY,
    device_id BIGINT NOT NULL,
    old_id BIGINT NOT NULL,
    payload TEXT NOT NULL,
    state VARCHAR(16) NOT NULL DEFAULT 'pending',
    attempts INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL,
    last_error VARCHAR(64) NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_telemetry_ingest_due
    ON telemetry_ingest_queue(state,next_attempt_at,id);
CREATE INDEX IF NOT EXISTS idx_telemetry_ingest_device_order
    ON telemetry_ingest_queue(device_id,id);
