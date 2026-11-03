-- Additive expansion: legacy readers retain their original column until backfill.
ALTER TABLE users
  ADD COLUMN user_phone_cipher text DEFAULT NULL,
  ADD COLUMN user_phone_lookup char(64) CHARACTER SET ascii COLLATE ascii_bin DEFAULT NULL,
  ADD COLUMN user_phone_protected boolean NOT NULL DEFAULT false,
  ADD CONSTRAINT uk_users_phone_lookup UNIQUE(user_phone_lookup),
  ADD INDEX ix_users_phone_protected(user_phone_protected),
  ADD CONSTRAINT ck_users_phone_storage CHECK (
    (user_phone_protected=false AND user_phone_cipher IS NULL AND user_phone_lookup IS NULL)
    OR (user_phone_protected=true AND user_phone IS NULL AND (
      (user_phone_cipher IS NULL AND user_phone_lookup IS NULL)
      OR (user_phone_cipher IS NOT NULL AND user_phone_lookup IS NOT NULL AND CHAR_LENGTH(user_phone_lookup)=64)
    ))
  );

CREATE TABLE IF NOT EXISTS account_phone_key_state (
  singleton tinyint PRIMARY KEY,
  lookup_verifier char(64) CHARACTER SET ascii COLLATE ascii_bin NOT NULL,
  CONSTRAINT ck_account_phone_singleton CHECK (singleton=1)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_bin;
