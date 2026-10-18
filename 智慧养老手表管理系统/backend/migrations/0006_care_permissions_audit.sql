-- 0006: Explicit staff duties and auditable access to sensitive care data.
-- Assignments are scoped to existing administrator accounts; FAMILY remains denied.
CREATE TABLE IF NOT EXISTS care_staff_permission (
  admin_id bigint PRIMARY KEY REFERENCES users(user_id) ON DELETE CASCADE,
  duty varchar(24) NOT NULL CHECK (duty IN ('supervisor','caregiver','auditor')),
  assigned_by bigint,
  assigned_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS care_access_audit (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  admin_id bigint NOT NULL,
  action varchar(64) NOT NULL,
  incident_id bigint,
  subject_admin_id bigint,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_care_access_audit_actor_time ON care_access_audit(admin_id,created_at DESC);
CREATE INDEX IF NOT EXISTS idx_care_access_audit_created ON care_access_audit(created_at);

-- Read/exports of personal, health and location data, written before disclosure.
CREATE TABLE IF NOT EXISTS care_sensitive_access_audit (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  actor_id bigint NOT NULL,
  actor_role varchar(16) NOT NULL,
  resource varchar(128) NOT NULL,
  method varchar(8) NOT NULL,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_care_sensitive_access_time
  ON care_sensitive_access_audit(created_at DESC);
CREATE INDEX IF NOT EXISTS idx_care_sensitive_access_actor
  ON care_sensitive_access_audit(actor_id,created_at DESC);

-- The worker updates this heartbeat after each polling cycle.
CREATE TABLE IF NOT EXISTS care_worker_heartbeat (
  singleton boolean PRIMARY KEY DEFAULT true CHECK (singleton),
  last_seen_at timestamptz NOT NULL,
  started_at timestamptz NOT NULL
);
