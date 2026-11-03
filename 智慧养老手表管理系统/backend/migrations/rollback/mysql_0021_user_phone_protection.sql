-- Preserve additive columns and all phone values. For an older application,
-- first stop writers and verify protect-user-phones --apply --rollback.
-- No decryption or destructive schema downgrade is performed by this SQL.
DELETE FROM schema_migrations WHERE name='mysql/0021_user_phone_protection.sql';
