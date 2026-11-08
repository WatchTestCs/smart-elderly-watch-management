-- PostgreSQL integration-compatible mirror. MySQL remains authoritative.
-- A receipt is committed with the business write, so a lost acknowledgement
-- can be replayed without a second write.
CREATE TABLE IF NOT EXISTS device_report_receipt (
    device_id BIGINT NOT NULL,
    binding_epoch BIGINT NOT NULL,
    request_id VARCHAR(64) NOT NULL,
    payload_digest CHAR(64) NOT NULL,
    result_json TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY(device_id,binding_epoch,request_id)
);
