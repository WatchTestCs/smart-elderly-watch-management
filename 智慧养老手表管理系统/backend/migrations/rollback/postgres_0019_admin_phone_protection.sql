-- Reverting protected administrator phones removes the encrypted columns and
-- key state. Backfill protected phones into the plaintext column and take a
-- backup before running this script because the encrypted values are lost.
-- PostgreSQL has no duplicate-column replay guard, so the objects are dropped
-- here to keep the additive migration reapplicable.
DROP TABLE IF EXISTS account_phone_key_state;
ALTER TABLE users
 DROP CONSTRAINT IF EXISTS uk_users_phone_lookup,
 DROP COLUMN IF EXISTS user_phone_protected,
 DROP COLUMN IF EXISTS user_phone_lookup,
 DROP COLUMN IF EXISTS user_phone_cipher;
DELETE FROM schema_migrations WHERE name='0019_admin_phone_protection.sql';
