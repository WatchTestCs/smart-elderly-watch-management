ALTER TABLE device ADD COLUMN health_epoch bigint NOT NULL DEFAULT 1;
CREATE TABLE health_clock (id integer PRIMARY KEY CHECK(id=1), revision bigint NOT NULL);
INSERT INTO health_clock VALUES(1,0);
CREATE TABLE health_sample (
  sample_id text PRIMARY KEY,
  old_id integer NOT NULL,
  device_id integer,
  epoch bigint NOT NULL,
  version bigint NOT NULL CHECK(version>0),
  revision bigint NOT NULL UNIQUE,
  payload jsonb NOT NULL
);
CREATE INDEX idx_health_owner ON health_sample(old_id,revision);
-- Clock updates serialize commits so a cursor cannot skip an uncommitted change.
CREATE TABLE health_change (
  revision bigint PRIMARY KEY,
  old_id integer NOT NULL,
  sample_id text NOT NULL,
  payload jsonb NOT NULL
);
CREATE INDEX idx_health_change_owner ON health_change(old_id,revision);

-- Cover all binding writers, including unbind and administrative updates.
CREATE FUNCTION advance_health_epoch() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
  IF NEW.old_id IS DISTINCT FROM OLD.old_id THEN
    NEW.health_epoch := OLD.health_epoch + 1;
  ELSE
    NEW.health_epoch := OLD.health_epoch;
  END IF;
  RETURN NEW;
END;
$$;
CREATE TRIGGER device_health_epoch BEFORE UPDATE ON device
FOR EACH ROW EXECUTE FUNCTION advance_health_epoch();
