package blob

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"suining-care/internal/db/dbtest"
	"suining-care/internal/securestore"
)

func TestMySQLImageStorageAndLegacyImport(t *testing.T) {
	pool := dbtest.MySQL(t)
	ctx := context.Background()
	box, err := securestore.New(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{8}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	raw, err := NewMySQL(pool.DB)
	if err != nil {
		t.Fatal(err)
	}
	manager := &Manager{DB: pool, Store: &Encrypted{Store: raw, Box: box}}
	if err := manager.Put(ctx, "current.png", []byte("current private image")); err != nil {
		t.Fatal(err)
	}
	if err := manager.Put(ctx, "current.png", []byte("overwrite")); !errors.Is(err, ErrExists) {
		t.Fatalf("immutable MySQL object overwritten: %v", err)
	}
	if got, err := manager.Get(ctx, "current.png"); err != nil || string(got) != "current private image" {
		t.Fatalf("encrypted MySQL round trip: %s, %v", got, err)
	}
	var encrypted []byte
	if err := pool.QueryRowContext(ctx, "SELECT payload FROM image_payload WHERE name='current.png'").Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encrypted, []byte("current private image")) {
		t.Fatal("plaintext persisted in MySQL image payload")
	}
	if _, err := raw.Get(ctx, "../current.png"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("invalid image key accepted: %v", err)
	}

	// Existing metadata may already name a legacy local object.
	original := []byte("existing local object")
	if _, err := pool.ExecContext(ctx, "INSERT INTO image_blob(name,content_sha256,size_bytes,state) VALUES(?,?,?,'ready')",
		"legacy.jpg", Digest(original), len(original)); err != nil {
		t.Fatal(err)
	}
	if missing, err := MissingMySQLReferences(ctx, pool.DB); err != nil || missing != 0 {
		t.Fatalf("unreferenced legacy catalog counted as missing: %d %v", missing, err)
	}
	if changed, err := ImportLegacy(ctx, pool.DB, box, "legacy.jpg", original); err != nil || !changed {
		t.Fatalf("legacy import: changed=%v err=%v", changed, err)
	}
	if changed, err := ImportLegacy(ctx, pool.DB, box, "legacy.jpg", original); err != nil || changed {
		t.Fatalf("idempotent retry: changed=%v err=%v", changed, err)
	}
	if got, err := manager.Get(ctx, "legacy.jpg"); err != nil || !bytes.Equal(got, original) {
		t.Fatalf("legacy bytes changed: %q %v", got, err)
	}
	if _, err := ImportLegacy(ctx, pool.DB, box, "legacy.jpg", []byte("different image")); !errors.Is(err, ErrIntegrity) {
		t.Fatalf("existing image was silently overwritten: %v", err)
	}
	if _, err := pool.ExecContext(ctx, "INSERT INTO pic(pic_id,pic_name) VALUES(?,?)", 9, "absent.png"); err != nil {
		t.Fatal(err)
	}
	if missing, err := MissingMySQLReferences(ctx, pool.DB); err != nil || missing != 1 {
		t.Fatalf("missing referenced image was not detected: %d %v", missing, err)
	}
	if _, err := ImportLegacy(ctx, pool.DB, box, "absent.png", []byte("restored")); err != nil {
		t.Fatal(err)
	}
	if missing, err := MissingMySQLReferences(ctx, pool.DB); err != nil || missing != 0 {
		t.Fatalf("restored reference still missing: %d %v", missing, err)
	}

	var entries int
	if err := raw.Walk(ctx, func(item Entry) error { entries++; return nil }); err != nil || entries != 3 {
		t.Fatalf("MySQL image walk: count=%d err=%v", entries, err)
	}
	if err := pool.QueryRowContext(ctx, "SELECT COUNT(*) FROM information_schema.TRIGGERS WHERE TRIGGER_SCHEMA=DATABASE() AND TRIGGER_NAME LIKE 'suining_analytics_%'").
		Scan(&entries); err != nil || entries != 0 {
		t.Fatalf("retired PostgreSQL outbox triggers still active: %d %v", entries, err)
	}
	// The native MySQL backend keeps the same backup-lock and GC contract.
	if _, err := pool.ExecContext(ctx, "DELETE FROM pic WHERE pic_id=9"); err != nil {
		t.Fatal(err)
	}
	if err := manager.AcquireBackup(ctx, "mysql-only-test", 60); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.Collect(ctx, time.Now().Add(time.Minute), false); !errors.Is(err, ErrBusy) {
		t.Fatalf("MySQL GC bypassed backup lock: %v", err)
	}
	if err := manager.ReleaseBackup(ctx, "mysql-only-test"); err != nil {
		t.Fatal(err)
	}
}
