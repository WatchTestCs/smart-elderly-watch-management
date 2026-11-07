-- 0022: Freeze legacy implicit supervisor access into explicit assignments.
-- After this one-time backfill, application authorization treats missing rows
-- as unassigned instead of silently granting supervisor privileges.
INSERT INTO care_staff_permission(admin_id,duty,assigned_by)
SELECT u.user_id,'supervisor',u.user_id
FROM users u
LEFT JOIN care_staff_permission p ON p.admin_id=u.user_id
WHERE u.role='1' AND p.admin_id IS NULL;
