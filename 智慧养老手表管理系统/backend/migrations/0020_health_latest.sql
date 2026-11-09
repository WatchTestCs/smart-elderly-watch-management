CREATE TABLE health_sample_latest (
 old_id bigint NOT NULL,
 series_key bytea NOT NULL,
 sample_id text NOT NULL,
 device_id integer,
 epoch bigint NOT NULL,
 revision bigint NOT NULL,
 payload jsonb NOT NULL,
 PRIMARY KEY(old_id,series_key)
);
CREATE TABLE health_latest_ready (
 old_id bigint PRIMARY KEY
);
CREATE INDEX idx_health_latest_recompute ON health_sample(old_id,source,source_device,metric,semantics,sample_start DESC);
