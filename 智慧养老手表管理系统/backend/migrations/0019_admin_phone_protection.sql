-- Match the protected administrator account contract in PostgreSQL-backed tests
-- and offline transfer databases. MySQL remains the operational primary.
ALTER TABLE users
 ADD COLUMN user_phone_cipher text,
 ADD COLUMN user_phone_lookup varchar(64) COLLATE "C",
 ADD COLUMN user_phone_protected boolean NOT NULL DEFAULT false,
 ADD CONSTRAINT uk_users_phone_lookup UNIQUE(user_phone_lookup);
CREATE TABLE account_phone_key_state (
 singleton smallint PRIMARY KEY CHECK(singleton=1),
 lookup_verifier varchar(64) COLLATE "C" NOT NULL
);
