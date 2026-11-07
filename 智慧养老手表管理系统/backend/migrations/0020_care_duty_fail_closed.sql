-- 0019: Freeze legacy implicit supervisor access into explicit assignments.
-- New administrators without a care_staff_permission row remain unassigned.
INSERT INTO care_staff_permission(admin_id,duty,assigned_by)
SELECT u.user_id,'supervisor',u.user_id
FROM users u
LEFT JOIN care_staff_permission p ON p.admin_id=u.user_id
WHERE u.role='1' AND p.admin_id IS NULL
ON CONFLICT(admin_id) DO NOTHING;
