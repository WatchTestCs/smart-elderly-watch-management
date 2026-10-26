CREATE TABLE IF NOT EXISTS image_blob (
 name text PRIMARY KEY,
 content_sha256 char(64) NOT NULL,
 size_bytes bigint NOT NULL,
 state varchar(16) NOT NULL DEFAULT 'ready',
 created_at timestamptz NOT NULL DEFAULT now()
);
CREATE INDEX IF NOT EXISTS idx_image_blob_gc ON image_blob(state,created_at);
CREATE TABLE IF NOT EXISTS image_gc_guard (
 id integer PRIMARY KEY,
 backup_owner varchar(64) NOT NULL DEFAULT '',
 backup_until timestamptz NOT NULL
);
INSERT INTO image_gc_guard(id,backup_owner,backup_until) VALUES(1,'','1970-01-01') ON CONFLICT DO NOTHING;
