package handlers

import (
	"bytes"
	"context"
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
	"testing"
	"time"

	"careold-go/internal/auth"
	"careold-go/internal/blob"
	"careold-go/internal/config"
	"careold-go/internal/csrf"
	"careold-go/internal/db"
	"careold-go/internal/securestore"
	"careold-go/internal/session"
	"github.com/go-sql-driver/mysql"
)

// Real S3-compatible server + real mysqldump/mysql restore. No mock object store
// and no queue-state edits. Only this test's disposable database/key are erased.
func TestImageS3TwoInstancesAndRealRestore(t *testing.T) {
	if os.Getenv("TEST_MYSQL_DSN") == "" {
		t.Skip("TEST_MYSQL_DSN not set; MySQL restore regression")
	}
	if os.Getenv("TEST_S3_ENDPOINT") == "" {
		t.Skip("TEST_S3_ENDPOINT not set; mandatory S3 integration gate")
	}
	pool := testDatabase(t)
	ctx := context.Background()
	seedDevice(t, pool)
	mustExec(t, pool, "INSERT INTO users(user_id,user_name) VALUES(1,'admin')")
	mustExec(t, pool, "INSERT INTO family(family_id,family_name,family_phone) VALUES(1,'synthetic','13800000001')")
	mustExec(t, pool, "UPDATE oldperson SET family_id=1 WHERE old_id=1")
	cfg := config.Config{DataKey: base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{6}, 32)), BlobBackend: "s3", Blob: blob.S3Options{Endpoint: os.Getenv("TEST_S3_ENDPOINT"), Region: "us-east-1", Bucket: os.Getenv("TEST_S3_BUCKET"), AccessKey: os.Getenv("TEST_S3_ACCESS_KEY"), SecretKey: os.Getenv("TEST_S3_SECRET_KEY")}}
	makeServer := func(pool *db.Database) (*httptest.Server, *session.Store, *Handler) {
		store := session.NewStore(time.Hour)
		t.Cleanup(store.Close)
		h := New(pool, store, cfg)
		store.UseDatabase(pool.SQL(), h.Box)
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
	if code := post(a.URL, "/api/v1/device/sos", map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "requestId": "restore-pending-sos"}); code != 200 {
		t.Fatalf("durable SOS fixture HTTP %d", code)
	}
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
	if err = pool.QueryRow(ctx, "SELECT DATABASE()").Scan(&database); err != nil {
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
	dump, err := client("mysqldump", []string{"--single-transaction", "--routines", "--triggers", "--no-tablespaces", "--set-gtid-purged=OFF", "--column-statistics=0", database}, nil)
	if err != nil {
		t.Fatalf("real backup failed: %v %s", err, dump)
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
	if output, err := client("mysql", []string{database}, dump); err != nil {
		t.Fatalf("real restore failed: %v %s", err, output)
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
	t.Cleanup(restored.Close)
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
	if err = restored.QueryRow(ctx, "SELECT count(*) FROM care_incident").Scan(&incidents); err != nil {
		t.Fatal(err)
	}
	if err = restored.QueryRow(ctx, "SELECT count(*) FROM care_notification_outbox WHERE status='pending'").Scan(&pending); err != nil {
		t.Fatal(err)
	}
	if err = restored.QueryRow(ctx, "SELECT count(*) FROM web_session").Scan(&sessions); err != nil {
		t.Fatal(err)
	}
	if incidents != 1 || pending != 1 || sessions != 1 {
		t.Fatalf("restored durable records incidents=%d pending=%d sessions=%d", incidents, pending, sessions)
	}
	if err = restoredManager.ReleaseBackup(ctx, owner); err != nil {
		t.Fatal(err)
	}
	t.Logf("LOCAL REAL MYSQL+S3 RESTORE image_sha256=%s mysql_dump_sha256=%s references=%d incidents=%d pending_outbox=%d sessions=%d; A stopped, B read; original DB and blob destroyed and restored; anonymous access denied", blob.Digest(original), blob.Digest(dump), len(references), incidents, pending, sessions)
}
