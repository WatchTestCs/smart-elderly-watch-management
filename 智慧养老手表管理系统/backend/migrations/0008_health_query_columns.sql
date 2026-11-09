-- Preserve immutable 0007 while adding portable, UTC-microsecond query fields.
ALTER TABLE health_sample ADD COLUMN source text;
ALTER TABLE health_sample ADD COLUMN source_device text;
ALTER TABLE health_sample ADD COLUMN metric text;
ALTER TABLE health_sample ADD COLUMN semantics text;
ALTER TABLE health_sample ADD COLUMN sample_start bigint;
ALTER TABLE health_sample ADD COLUMN sample_end bigint;
ALTER TABLE health_sample ADD COLUMN deleted boolean;
UPDATE health_sample SET source=payload->>'source',source_device=payload->>'sourceDevice',
 metric=payload->>'metric',semantics=payload->>'semantics',
 sample_start=floor(extract(epoch FROM (payload->>'start')::timestamptz)*1000000)::bigint,
 sample_end=floor(extract(epoch FROM (payload->>'end')::timestamptz)*1000000)::bigint,
 deleted=COALESCE((payload->>'deleted')::boolean,false);
ALTER TABLE health_sample ALTER COLUMN source SET NOT NULL;
ALTER TABLE health_sample ALTER COLUMN source_device SET NOT NULL;
ALTER TABLE health_sample ALTER COLUMN metric SET NOT NULL;
ALTER TABLE health_sample ALTER COLUMN semantics SET NOT NULL;
ALTER TABLE health_sample ALTER COLUMN sample_start SET NOT NULL;
ALTER TABLE health_sample ALTER COLUMN sample_end SET NOT NULL;
ALTER TABLE health_sample ALTER COLUMN deleted SET NOT NULL;
CREATE INDEX idx_health_time ON health_sample(old_id,sample_start);
