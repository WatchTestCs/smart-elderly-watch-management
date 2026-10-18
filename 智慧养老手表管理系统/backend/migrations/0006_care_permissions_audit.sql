-- 0006: Explicit staff duties and auditable access to sensitive care data.
-- Assignments are scoped to existing administrator accounts; FAMILY remains denied.
CREATE TABLE IF NOT EXISTS care_staff_permission (
  admin_id bigint PRIMARY KEY,
  duty varchar(24) NOT NULL CHECK (duty IN ('supervisor','caregiver','auditor')),
  assigned_by bigint,
  assigned_at timestamptz NOT NULL DEFAULT now()
);
CREATE TABLE IF NOT EXISTS care_access_audit (
  id bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
  admin_id bigint NOT NULL,
  action varchar(64) NOT NULL,
  incident_id bigint,
  created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_care_access_audit_actor_time ON care_access_audit(admin_id,created_at DESC);
CREATE INDEX IF NOT EXISTS idx_care_access_audit_created ON care_access_audit(created_at);
