ALTER TABLE users
  ADD COLUMN user_phone_cipher text,
  ADD COLUMN user_phone_lookup varchar(64) COLLATE "C",
  ADD COLUMN user_phone_protected boolean NOT NULL DEFAULT false,
  ADD CONSTRAINT uk_users_phone_lookup UNIQUE(user_phone_lookup),
  ADD CONSTRAINT ck_users_phone_storage CHECK (
    (user_phone_protected=false AND user_phone_cipher IS NULL AND user_phone_lookup IS NULL)
    OR (user_phone_protected=true AND user_phone IS NULL AND (
      (user_phone_cipher IS NULL AND user_phone_lookup IS NULL)
      OR (user_phone_cipher IS NOT NULL AND user_phone_lookup IS NOT NULL AND CHAR_LENGTH(user_phone_lookup)=64)
    ))
  );

CREATE INDEX ix_users_phone_protected ON users(user_phone_protected);

CREATE TABLE account_phone_key_state (
  singleton smallint PRIMARY KEY CHECK (singleton=1),
  lookup_verifier varchar(64) COLLATE "C" NOT NULL
);
