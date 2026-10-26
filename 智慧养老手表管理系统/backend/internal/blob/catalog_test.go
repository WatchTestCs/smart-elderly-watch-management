package blob

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"testing"
	"time"

	"careold-go/internal/db/dbtest"
	"careold-go/internal/securestore"
)

func TestCatalogGCReferenceLocksAndBackupLease(t *testing.T) {
	pool := dbtest.MySQL(t)
	ctx := context.Background()
	local, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	box, _ := securestore.New(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{1}, 32)))
	m := &Manager{DB: pool, Store: &Encrypted{Store: local, Box: box}}
	if err = m.Put(ctx, "retained.png", []byte("private")); err != nil {
		t.Fatal(err)
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err = m.Require(ctx, tx, []string{"retained.png"}); err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, "INSERT INTO pic(pic_id,pic_name) VALUES(1,'retained.png')"); err != nil {
		t.Fatal(err)
	}
	result := make(chan error, 1)
	go func() {
		count, err := m.Collect(ctx, time.Now().Add(time.Minute), false)
		if count != 0 && err == nil {
			err = errors.New("referenced image collected")
		}
		result <- err
	}()
	if err = tx.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	if err = <-result; err != nil {
		t.Fatal(err)
	}
	if _, err = m.Get(ctx, "retained.png"); err != nil {
		t.Fatal(err)
	}
	if _, err = pool.Exec(ctx, "DELETE FROM pic WHERE pic_id=1"); err != nil {
		t.Fatal(err)
	}
	if err = m.AcquireBackup(ctx, "test-backup", 60); err != nil {
		t.Fatal(err)
	}
	if _, err = m.Collect(ctx, time.Now().Add(time.Minute), false); !errors.Is(err, ErrBusy) {
		t.Fatalf("GC ignored backup lease: %v", err)
	}
	if err = m.ReleaseBackup(ctx, "test-backup"); err != nil {
		t.Fatal(err)
	}
	if count, err := m.Collect(ctx, time.Now().Add(time.Minute), false); err != nil || count != 1 {
		t.Fatal("orphan not collected", count, err)
	}
	if _, err = m.Get(ctx, "retained.png"); !errors.Is(err, ErrNotFound) {
		t.Fatal("deleted image still readable", err)
	}
	tx, err = pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer tx.Rollback(ctx)
	if err = m.Require(ctx, tx, []string{"retained.png"}); !errors.Is(err, ErrNotFound) {
		t.Fatal("deleted image rebound", err)
	}
}

func TestCatalogInsertFailureCompensatesBlob(t *testing.T) {
	pool := dbtest.MySQL(t)
	ctx := context.Background()
	local, err := NewLocal(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	m := &Manager{DB: pool, Store: local}
	if _, err = pool.Exec(ctx, "DROP TABLE image_blob"); err != nil {
		t.Fatal(err)
	}
	if err = m.Put(ctx, "failed.png", []byte("synthetic")); err == nil {
		t.Fatal("missing catalog reported success")
	}
	if _, err = local.Get(ctx, "failed.png"); !errors.Is(err, ErrNotFound) {
		t.Fatal("failed catalog insert left bytes", err)
	}
}
