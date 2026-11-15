package handlers

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"image"
	"image/png"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-sql-driver/mysql"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"suining-care/internal/analytics"
	"suining-care/internal/auth"
	"suining-care/internal/blob"
	"suining-care/internal/config"
	"suining-care/internal/csrf"
	"suining-care/internal/db"
	"suining-care/internal/securestore"
	"suining-care/internal/session"
)

// Real S3-compatible server + real mysqldump/mysql restore. No mock object store
// and no queue-state edits. The key backup is separate from the SQL/object
// backup. Rebuild the lost PG projection from the restored MySQL source.
// Only this test's disposable databases, objects and key are affected.
func TestImageS3TwoInstancesAndRealRestore(t *testing.T) {
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		t.Skip("TEST_MYSQL_DSN not set; MySQL restore regression")
	}
	if os.Getenv("TEST_S3_ENDPOINT") == "" {
		t.Skip("TEST_S3_ENDPOINT not set; mandatory S3 integration gate")
	}
	if os.Getenv("TEST_DATABASE_URL") == "" {
		t.Skip("TEST_DATABASE_URL not set; coordinated projection restore requires PostgreSQL")
	}
	pool := testDatabase(t)
	ctx := context.Background()
	seedDevice(t, pool)
	seedCareAdminDuty(t, pool, 1, "supervisor")
	mustExec(t, pool, "INSERT INTO family(family_id,family_name,family_phone) VALUES(1,'synthetic','13800000001')")
	mustExec(t, pool, "UPDATE oldperson SET family_id=1 WHERE old_id=1")
	cfg := config.Config{DataKey: base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{6}, 32)), BlobBackend: "s3", Blob: blob.S3Options{Endpoint: os.Getenv("TEST_S3_ENDPOINT"), Region: "us-east-1", Bucket: os.Getenv("TEST_S3_BUCKET"), AccessKey: os.Getenv("TEST_S3_ACCESS_KEY"), SecretKey: os.Getenv("TEST_S3_SECRET_KEY")}}
	keyBackup := filepath.Join(t.TempDir(), "keyring")
	if err := os.WriteFile(keyBackup, []byte(cfg.DataKey), 0600); err != nil {
		t.Fatal("could not preserve the separate test key backup")
	}
	makeServer := func(pool *db.Database) (*httptest.Server, *session.Store, *Handler) {
		store := session.NewStore(time.Hour)
		t.Cleanup(store.Close)
		h := New(pool, store, cfg)
		store.UseDatabase(pool.DB, h.Box)
		mux := http.NewServeMux()
		h.Routes(mux)
		server := httptest.NewServer(csrf.Middleware(store, auth.Middleware(store, h.EnforceAdminDuty(mux))))
		t.Cleanup(server.Close)
		return server, store, h
	}
	a, storeA, hA := makeServer(pool)
	b, storeB, _ := makeServer(pool)
	sess := storeA.Create(session.Principal{Role: auth.RoleAdmin, ID: 1})
	if sess == nil {
		t.Fatal("session persistence failed")
	}
	var form bytes.Buffer
	writer := multipart.NewWriter(&form)
	field, err := writer.CreateFormFile("file", "synthetic.png")
	if err != nil {
		t.Fatal(err)
	}
	if err = png.Encode(field, image.NewRGBA(image.Rect(0, 0, 2, 2))); err != nil {
		t.Fatal(err)
	}
	if err = writer.Close(); err != nil {
		t.Fatal(err)
	}
	request, _ := http.NewRequest(http.MethodPost, a.URL+"/api/v1/image/saveImage", &form)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	request.Header.Set(csrf.HeaderName, sess.CSRFToken)
	request.AddCookie(&http.Cookie{Name: session.CookieName, Value: sess.ID})
	response, err := http.DefaultClient.Do(request)
	if err != nil {
		t.Fatal(err)
	}
	var upload struct {
		Filename string `json:"filename"`
	}
	decodeErr := json.NewDecoder(response.Body).Decode(&upload)
	response.Body.Close()
	if response.StatusCode != 200 || decodeErr != nil || upload.Filename == "" {
		t.Fatalf("upload HTTP %d, %v", response.StatusCode, decodeErr)
	}
	key := upload.Filename
	post := func(server, path string, body any) int {
		data, _ := json.Marshal(body)
		req, _ := http.NewRequest(http.MethodPost, server+path, bytes.NewReader(data))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(csrf.HeaderName, sess.CSRFToken)
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: sess.ID})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		return resp.StatusCode
	}
	if code := post(a.URL, "/api/v1/family/updateTouPic", map[string]any{"familyPhone": "13800000001", "familyTouPic": key}); code != 200 {
		t.Fatalf("image reference HTTP %d", code)
	}
	if code := post(a.URL, "/api/v1/device/sos", map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "requestId": "restore-pending-sos", "bindingVersion": 1}); code != 200 {
		t.Fatalf("durable SOS fixture HTTP %d", code)
	}
	mustExec(t, pool, "INSERT INTO ratepre(id,device_id,old_id,hr,create_time) VALUES(1,1,1,72,CURRENT_TIMESTAMP(6))")
	mustExec(t, pool, "INSERT INTO care_access_audit(admin_id,action,incident_id) VALUES(1,'restore-fixture',1)")
	pg := restoreProjectionDatabase(t)
	drainRestoredProjection(t, pool, pg)
	projectionBefore := projectionDigest(t, pg)
	read := func(server string) []byte {
		req, _ := http.NewRequest(http.MethodGet, server+"/api/v1/image/download?name="+key, nil)
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: sess.ID})
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		data, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil || resp.StatusCode != 200 {
			t.Fatalf("cross-instance image HTTP %d: %v", resp.StatusCode, err)
		}
		return data
	}
	original := read(b.URL)
	a.Close()
	storeA.Close()
	if !bytes.Equal(read(b.URL), original) {
		t.Fatal("B lost image after A stopped")
	}
	raw, err := blob.NewS3(cfg.Blob)
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := raw.Get(ctx, key)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(sealed, original) || !bytes.HasPrefix(sealed, []byte(securestore.Prefix)) {
		t.Fatal("private bucket stored plaintext image")
	}
	wrongKey, err := securestore.New(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{7}, 32)))
	if err != nil {
		t.Fatal(err)
	}
	if plain, err := wrongKey.Open(string(sealed), "image:"+key); err == nil || plain != "" {
		t.Fatal("a different recovery key exposed the protected image")
	}
	anon, err := http.Get(cfg.Blob.Endpoint + "/" + cfg.Blob.Bucket + "/suining-images/" + key)
	if err != nil {
		t.Fatal(err)
	}
	io.Copy(io.Discard, anon.Body)
	anon.Body.Close()
	if anon.StatusCode != 403 && anon.StatusCode != 401 {
		t.Fatalf("bucket anonymously readable: HTTP %d", anon.StatusCode)
	}
	manager, err := hA.images()
	if err != nil {
		t.Fatal(err)
	}
	owner := "integration-restore"
	if err = manager.AcquireBackup(ctx, owner, 300); err != nil {
		t.Fatal(err)
	}
	backupDir := t.TempDir()
	if err = os.WriteFile(filepath.Join(backupDir, key), sealed, 0600); err != nil {
		t.Fatal(err)
	}
	dsn, err := mysql.ParseDSN(os.Getenv("TEST_MYSQL_DSN"))
	if err != nil {
		t.Fatal(err)
	}
	host, port, err := net.SplitHostPort(dsn.Addr)
	if err != nil {
		t.Fatal(err)
	}
	var database string
	if err = pool.QueryRowContext(ctx, "SELECT DATABASE()").Scan(&database); err != nil {
		t.Fatal(err)
	}
	client := func(name string, args []string, input []byte) ([]byte, error) {
		path, err := exec.LookPath(name)
		if err != nil {
			return nil, err
		}
		command := exec.Command(path, append([]string{"--no-defaults", "--protocol=TCP", "--host=" + host, "--port=" + port, "--user=" + dsn.User}, args...)...)
		command.Env = append(os.Environ(), "MYSQL_PWD="+dsn.Passwd)
		command.Stdin = bytes.NewReader(input)
		return command.CombinedOutput()
	}
	before := recoveryDataDigests(t, pool)
	dump, err := client("mysqldump", []string{"--single-transaction", "--routines", "--triggers", "--no-tablespaces", "--set-gtid-purged=OFF", "--column-statistics=0", database}, nil)
	if err != nil {
		// Combined output can include a partial SQL dump; never publish it.
		t.Fatalf("real backup failed: %v (output redacted)", err)
	}
	if err = os.WriteFile(filepath.Join(backupDir, "mysql.sql"), dump, 0600); err != nil {
		t.Fatal(err)
	}
	if err = manager.ReleaseBackup(ctx, owner); err != nil {
		t.Fatal(err)
	}
	b.Close()
	storeB.Close()
	pool.Close()
	cfg.DataKey = ""
	if _, err = pg.Exec(ctx, "DROP SCHEMA suining_analytics CASCADE"); err != nil {
		t.Fatal(err)
	}
	// Destroy only the disposable source database, then rebuild from its dump.
	if output, err := client("mysql", []string{"-e", "DROP DATABASE `" + database + "`; CREATE DATABASE `" + database + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_bin"}, nil); err != nil {
		t.Fatalf("isolated destruction failed: %v %s", err, output)
	}
	if err = raw.Delete(ctx, key); err != nil {
		t.Fatal(err)
	}
	if _, err = raw.Get(ctx, key); !errors.Is(err, blob.ErrNotFound) {
		t.Fatal("test object was not actually removed", err)
	}
	if _, err := client("mysql", []string{database}, dump); err != nil {
		t.Fatalf("real restore failed: %v (SQL output redacted)", err)
	}
	backupBlob, err := os.ReadFile(filepath.Join(backupDir, key))
	if err != nil {
		t.Fatal(err)
	}
	if err = raw.Put(ctx, key, backupBlob); err != nil {
		t.Fatal(err)
	}
	dsn.DBName = database
	restored, err := db.OpenMySQL(ctx, dsn.FormatDSN())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = restored.Close() })
	for name, digest := range recoveryDataDigests(t, restored) {
		if digest != before[name] {
			t.Fatalf("restored %s differs from its coordinated snapshot", name)
		}
	}
	if err = db.MigrateMySQL(ctx, restored); err != nil {
		t.Fatal("restored schema failed current migration validation", err)
	}
	if recoveryDataDigests(t, restored)["migrations"] != before["migrations"] {
		t.Fatal("migration validation changed the restored schema history")
	}
	recoveredKey, err := os.ReadFile(keyBackup)
	if err != nil {
		t.Fatal("the separate recovery key is unavailable")
	}
	cfg.DataKey = string(recoveredKey)
	c, _, hC := makeServer(restored)
	if !bytes.Equal(read(c.URL), original) {
		t.Fatal("restored image differs")
	}
	restoredManager, err := hC.images()
	if err != nil {
		t.Fatal(err)
	}
	references, err := restoredManager.VerifyReferences(ctx)
	if err != nil || len(references) != 1 {
		t.Fatalf("dangling restored reference: %v %v", references, err)
	}
	var incidents, pending, sessions int
	if err = restored.QueryRowContext(ctx, "SELECT count(*) FROM care_incident").Scan(&incidents); err != nil {
		t.Fatal(err)
	}
	if err = restored.QueryRowContext(ctx, "SELECT count(*) FROM care_notification_outbox WHERE status='pending'").Scan(&pending); err != nil {
		t.Fatal(err)
	}
	if err = restored.QueryRowContext(ctx, "SELECT count(*) FROM web_session").Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if incidents != 1 || pending != 1 || sessions != 1 {
		t.Fatalf("restored durable records incidents=%d pending=%d sessions=%d", incidents, pending, sessions)
	}
	if err = analytics.Migrate(ctx, pg); err != nil {
		t.Fatal(err)
	}
	var queued int
	if err = restored.QueryRowContext(ctx, "SELECT count(*) FROM analytics_outbox").Scan(&queued); err != nil || queued != 0 {
		t.Fatalf("snapshot was not drained before projection loss: %d %v", queued, err)
	}
	if n, err := analytics.Backfill(ctx, restored); err != nil || n < 3 {
		t.Fatalf("could not rebuild projection from restored primary: %d %v", n, err)
	}
	drainRestoredProjection(t, restored, pg)
	if projectionDigest(t, pg) != projectionBefore {
		t.Fatal("rebuilt PostgreSQL projection differs from the restored MySQL source")
	}
	// A repeated rebuild must remain idempotent, including source identity.
	if _, err = analytics.Backfill(ctx, restored); err != nil {
		t.Fatal(err)
	}
	drainRestoredProjection(t, restored, pg)
	if projectionDigest(t, pg) != projectionBefore {
		t.Fatal("repeated projection rebuild changed logical rows")
	}
	if code := post(c.URL, "/api/v1/device/sos", map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "requestId": "restore-pending-sos", "bindingVersion": 1}); code != 200 {
		t.Fatalf("restored application rejected a lost-ack SOS retry: HTTP %d", code)
	}
	if err = restored.QueryRowContext(ctx, "SELECT count(*) FROM care_incident").Scan(&incidents); err != nil || incidents != 1 {
		t.Fatalf("restored retry duplicated the incident: %d %v", incidents, err)
	}
	if err = restored.QueryRowContext(ctx, "SELECT count(*) FROM care_notification_outbox").Scan(&pending); err != nil || pending != 1 {
		t.Fatalf("restored retry duplicated the notification: %d %v", pending, err)
	}
	if err = restoredManager.ReleaseBackup(ctx, owner); err != nil {
		t.Fatal(err)
	}
	t.Logf("LOCAL REAL MYSQL+S3+PG RESTORE image_sha256=%s mysql_dump_sha256=%s references=%d incidents=%d pending_outbox=%d sessions=%d; independent key restored, wrong key rejected; migration/health/audit/queue digests preserved; lost PG projection rebuilt twice; SOS retry remained idempotent", blob.Digest(original), blob.Digest(dump), len(references), incidents, pending, sessions)
}

// Hash exact ordered rows without putting sessions, health or audit payloads in
// logs. The fixed query list is test-owned and cannot accept external SQL.
func recoveryDataDigests(t *testing.T, pool *db.Database) map[string][32]byte {
	t.Helper()
	queries := map[string]string{
		"migrations": "SELECT name,checksum,schema_checksum,applied_at FROM schema_migrations ORDER BY name",
		"health":     "SELECT * FROM ratepre ORDER BY id",
		"incidents":  "SELECT * FROM care_incident ORDER BY id",
		"events":     "SELECT * FROM care_incident_event ORDER BY id",
		"audit":      "SELECT * FROM care_access_audit ORDER BY id",
		"queue":      "SELECT * FROM care_notification_outbox ORDER BY id",
		"sessions":   "SELECT * FROM web_session ORDER BY id_hash",
	}
	result := make(map[string][32]byte, len(queries))
	for name, query := range queries {
		rows, err := pool.QueryContext(context.Background(), query)
		if err != nil {
			t.Fatal(err)
		}
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			t.Fatal(err)
		}
		var records [][]any
		for rows.Next() {
			values := make([]any, len(columns))
			targets := make([]any, len(columns))
			for i := range values {
				targets[i] = &values[i]
			}
			if err = rows.Scan(targets...); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			records = append(records, values)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			t.Fatal(err)
		}
		if len(records) == 0 {
			t.Fatalf("recovery fixture has no %s rows", name)
		}
		body, err := json.Marshal(records)
		if err != nil {
			t.Fatal(err)
		}
		result[name] = sha256.Sum256(body)
	}
	return result
}

func restoreProjectionDatabase(t *testing.T) *pgxpool.Pool {
	t.Helper()
	ctx := context.Background()
	dsn := os.Getenv("TEST_DATABASE_URL")
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	name := "restore_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	identifier := pgx.Identifier{name}.Sanitize()
	if _, err = admin.Exec(ctx, "CREATE DATABASE "+identifier); err != nil {
		t.Fatal(err)
	}
	var pg *pgxpool.Pool
	t.Cleanup(func() {
		if pg != nil {
			pg.Close()
		}
		if _, err := admin.Exec(ctx, "DROP DATABASE "+identifier+" WITH (FORCE)"); err != nil {
			t.Error(err)
		}
	})
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.Database = name
	pg, err = pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err = analytics.Migrate(ctx, pg); err != nil {
		t.Fatal(err)
	}
	return pg
}

func drainRestoredProjection(t *testing.T, primary *db.Database, pg *pgxpool.Pool) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	for {
		n, err := analytics.SyncBatch(ctx, primary, pg, 250)
		if err != nil {
			t.Fatal(err)
		}
		if n == 0 {
			return
		}
	}
}

func projectionDigest(t *testing.T, pg *pgxpool.Pool) [32]byte {
	t.Helper()
	var content string
	err := pg.QueryRow(context.Background(), `SELECT COALESCE(jsonb_agg(
		jsonb_build_array(source_id::text,entity,entity_id,payload,deleted)
		ORDER BY source_id,entity,entity_id),'[]'::jsonb)::text FROM suining_analytics.record`).Scan(&content)
	if err != nil || content == "[]" {
		t.Fatalf("projection fixture is unavailable or empty: %v", err)
	}
	return sha256.Sum256([]byte(content))
}
