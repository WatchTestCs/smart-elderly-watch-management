CREATE TABLE IF NOT EXISTS device_request_nonces (
    device_id BIGINT NOT NULL,
    nonce_digest BYTEA NOT NULL,
    expires_at TIMESTAMPTZ NOT NULL,
    PRIMARY KEY (device_id, nonce_digest),
    CONSTRAINT device_nonce_digest_length CHECK (octet_length(nonce_digest) = 32)
);
CREATE INDEX IF NOT EXISTS idx_device_request_nonce_expiry
    ON device_request_nonces(expires_at);
