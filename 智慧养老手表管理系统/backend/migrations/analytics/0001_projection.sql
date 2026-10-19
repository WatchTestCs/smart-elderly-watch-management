CREATE SCHEMA IF NOT EXISTS suining_analytics;
CREATE TABLE IF NOT EXISTS suining_analytics.record (
 source_id uuid NOT NULL, entity varchar(32) NOT NULL, entity_id bigint NOT NULL,
 last_event_id bigint NOT NULL, payload jsonb, deleted boolean NOT NULL,
 synced_at timestamptz NOT NULL DEFAULT now(),
 PRIMARY KEY(source_id,entity,entity_id)
);
CREATE INDEX IF NOT EXISTS analytics_record_entity ON suining_analytics.record(source_id,entity) WHERE NOT deleted;
CREATE OR REPLACE VIEW suining_analytics.daily_health AS
 SELECT source_id,(payload->>'create_time')::timestamp::date AS day,
 count(*) AS reports,
 count((payload->>'hr')::int) AS heart_rate_samples,
 count((payload->>'bp')::int) AS blood_pressure_samples,
 avg((payload->>'hr')::int) AS mean_heart_rate,
 avg((payload->>'bp')::int) AS mean_blood_pressure,
 count(*) FILTER (WHERE payload->>'rate_state'='1') AS heart_rate_alerts,
 count(*) FILTER (WHERE payload->>'pre_state'='1') AS blood_pressure_alerts
 FROM suining_analytics.record WHERE entity='ratepre' AND NOT deleted
 GROUP BY source_id,(payload->>'create_time')::timestamp::date;
CREATE OR REPLACE VIEW suining_analytics.care_summary AS
 SELECT source_id,payload->>'state' AS state,count(*) AS incidents
 FROM suining_analytics.record WHERE entity='care_incident' AND NOT deleted
 GROUP BY source_id,payload->>'state';
