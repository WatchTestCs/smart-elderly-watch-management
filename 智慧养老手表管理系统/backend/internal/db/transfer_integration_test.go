package db_test

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"suining-care/internal/db"
	"suining-care/internal/db/dbtest"
)

func TestLegacyTransferIsCheckedAndAtomic(t *testing.T) {
	for _, version := range []int{6, 7, 8} {
		t.Run(fmt.Sprintf("PG000%d", version), func(t *testing.T) { testLegacyTransfer(t, version) })
	}
}

func testLegacyTransfer(t *testing.T, version int) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set")
	}
	target := dbtest.MySQL(t)
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	name := "transfer_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+name); err != nil {
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(url)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = name
	source, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { source.Close(); admin.Exec(ctx, "DROP SCHEMA "+name+" CASCADE"); admin.Close() })
	if err = db.Migrate(ctx, source); err != nil {
		t.Fatal(err)
	}
	if version < 8 {
		if _, err = source.Exec(ctx, "ALTER TABLE health_sample DROP COLUMN source,DROP COLUMN source_device,DROP COLUMN metric,DROP COLUMN semantics,DROP COLUMN sample_start,DROP COLUMN sample_end,DROP COLUMN deleted"); err != nil {
			t.Fatal(err)
		}
	}
	if version == 6 {
		if _, err = source.Exec(ctx, "DROP TABLE health_sample,health_change,health_clock; DROP TRIGGER device_health_epoch ON device; DROP FUNCTION advance_health_epoch(); ALTER TABLE device DROP COLUMN health_epoch"); err != nil {
			t.Fatal(err)
		}
	}
	for _, query := range []string{
		"INSERT INTO users(user_id,user_name) VALUES(7,'主管')",
		"UPDATE app_metadata SET schema_version=8 WHERE id=1",
		"INSERT INTO oldperson(old_id,name) VALUES(11,'长者')",
		"INSERT INTO device(device_id,device_no,secret,old_id) VALUES(5,'IMPORT','hashed-secret',11)",
		"UPDATE oldperson SET device_id=5 WHERE old_id=11",
		"INSERT INTO ratepre(id,device_id,old_id,hr,bp) VALUES(9,5,11,72,NULL)",
		"INSERT INTO image_blob(name,content_sha256,size_bytes,state) VALUES('elder/avatar.png',repeat('a',64),128,'ready')",
		"INSERT INTO device_request_nonces(device_id,nonce_digest,expires_at) VALUES(5,decode(repeat('ab',32),'hex'),now()+interval '10 minutes')",
	} {
		if _, err = source.Exec(ctx, query); err != nil {
			t.Fatal(err)
		}
	}
	if version >= 7 {
		payload := `{"recordId":"imported","source":"watch","sourceDevice":"IMPORT","metric":"heart_rate","unit":"count/min","value":72,"start":"2026-10-19T09:00:00+08:00","end":"2026-10-19T09:00:00+08:00","semantics":"instant","version":2}`
		id := strings.Repeat("a", 64)
		query := "INSERT INTO health_sample(sample_id,old_id,device_id,epoch,version,revision,payload) VALUES($1,11,5,3,2,4,$2)"
		if version == 8 {
			query = "INSERT INTO health_sample(sample_id,old_id,device_id,epoch,version,revision,payload,source,source_device,metric,semantics,sample_start,sample_end,deleted) VALUES($1,11,5,3,2,4,$2,'watch','IMPORT','heart_rate','instant',1792371600000000,1792371600000000,false)"
		}
		if _, err = source.Exec(ctx, query, id, payload); err != nil {
			t.Fatal(err)
		}
		if _, err = source.Exec(ctx, "UPDATE health_clock SET revision=4; UPDATE device SET old_id=NULL; UPDATE device SET old_id=11"); err != nil {
			t.Fatal(err)
		}
		if _, err = source.Exec(ctx, "INSERT INTO health_change VALUES(4,11,$1,$2)", id, payload); err != nil {
			t.Fatal(err)
		}
	}
	bindingEpoch := int64(1)
	if version >= 7 {
		bindingEpoch = 3
	}
	if _, err = source.Exec(ctx, "INSERT INTO telemetry_inbox(device_id,elder_id,binding_epoch,request_id,payload,status,attempts,next_retry_at,created_at) VALUES(5,11,$1,'queued-import','sealed-payload','pending',2,now()+interval '30 seconds',now())", bindingEpoch); err != nil {
		t.Fatal(err)
	}
	counts, err := db.TransferLegacy(ctx, source, target, false)
	if err != nil || counts["ratepre"] != 1 || counts["app_metadata"] != 1 ||
		counts["image_blob"] != 1 || counts["device_request_nonces"] != 1 ||
		counts["telemetry_inbox"] != 1 {
		t.Fatalf("preflight: %v %v", counts, err)
	}
	var count int
	if err = target.QueryRow(ctx, "SELECT count(*) FROM users").Scan(&count); err != nil || count != 0 {
		t.Fatal("preflight wrote data")
	}
	var bootstrapVersion int
	if err = target.QueryRow(ctx, "SELECT schema_version FROM app_metadata WHERE id=1").Scan(&bootstrapVersion); err != nil || bootstrapVersion != 1 {
		t.Fatalf("preflight changed target bootstrap metadata: version=%d err=%v", bootstrapVersion, err)
	}
	if _, err = db.TransferLegacy(ctx, source, target, true); err != nil {
		t.Fatal(err)
	}
	var nameRead string
	var hr int
	var bp *int
	if err = target.QueryRow(ctx, "SELECT name FROM oldperson WHERE old_id=11").Scan(&nameRead); err != nil || nameRead != "长者" {
		t.Fatalf("identity lost: %s %v", nameRead, err)
	}
	if err = target.QueryRow(ctx, "SELECT hr,bp FROM ratepre WHERE id=9").Scan(&hr, &bp); err != nil || hr != 72 || bp != nil {
		t.Fatalf("metrics changed: %d %v %v", hr, bp, err)
	}
	if version >= 7 {
		var start, revision, epoch int64
		var sourceName string
		if err = target.QueryRow(ctx, "SELECT sample_start,source FROM health_sample").Scan(&start, &sourceName); err != nil || start != 1792371600000000 || sourceName != "watch" {
			t.Fatalf("sample query fields lost: %d %s %v", start, sourceName, err)
		}
		var samplePayload []byte
		if err = target.QueryRow(ctx, "SELECT payload FROM health_sample").Scan(&samplePayload); err != nil {
			t.Fatal(err)
		}
		var sample map[string]any
		if err = json.Unmarshal(samplePayload, &sample); err != nil || sample["recordId"] != "imported" || sample["sourceDevice"] != "IMPORT" {
			t.Fatalf("health_sample JSON payload corrupted: %s %v", samplePayload, err)
		}
		var changePayload []byte
		if err = target.QueryRow(ctx, "SELECT payload FROM health_change").Scan(&changePayload); err != nil {
			t.Fatal(err)
		}
		var change map[string]any
		if err = json.Unmarshal(changePayload, &change); err != nil || change["recordId"] != "imported" {
			t.Fatalf("health_change JSON payload corrupted: %s %v", changePayload, err)
		}
		if err = target.QueryRow(ctx, "SELECT revision FROM health_clock WHERE id=1").Scan(&revision); err != nil || revision != 4 {
			t.Fatalf("clock lost: %d %v", revision, err)
		}
		if err = target.QueryRow(ctx, "SELECT health_epoch FROM device WHERE device_id=5").Scan(&epoch); err != nil || epoch != 3 {
			t.Fatalf("binding epoch lost: %d %v", epoch, err)
		}
	}
	var metadataVersion int
	if err = target.QueryRow(ctx, "SELECT schema_version FROM app_metadata WHERE id=1").Scan(&metadataVersion); err != nil || metadataVersion != 8 {
		t.Fatalf("app metadata lost: %d %v", metadataVersion, err)
	}
	var imageName string
	if err = target.QueryRow(ctx, "SELECT name FROM image_blob WHERE name='elder/avatar.png'").Scan(&imageName); err != nil || imageName != "elder/avatar.png" {
		t.Fatalf("image blob metadata lost: %s %v", imageName, err)
	}
	var nonce []byte
	if err = target.QueryRow(ctx, "SELECT nonce_digest FROM device_request_nonces WHERE device_id=5").Scan(&nonce); err != nil || len(nonce) != 32 {
		t.Fatalf("device replay state lost: %d %v", len(nonce), err)
	}
	var queuePayload, queueStatus string
	if err = target.QueryRow(ctx, "SELECT payload,status FROM telemetry_inbox WHERE request_id='queued-import'").Scan(&queuePayload, &queueStatus); err != nil || queuePayload != "sealed-payload" || queueStatus != "pending" {
		t.Fatalf("durable telemetry inbox lost: %s %s %v", queuePayload, queueStatus, err)
	}
	if _, err = db.TransferLegacy(ctx, source, target, true); err == nil {
		t.Fatal("nonempty target accepted")
	}
	failed := dbtest.MySQL(t)
	if _, err = failed.SQL().ExecContext(ctx, "CREATE TRIGGER reject_transfer BEFORE INSERT ON ratepre FOR EACH ROW SIGNAL SQLSTATE '45000' SET MESSAGE_TEXT='test rejection'"); err != nil {
		t.Fatal(err)
	}
	if _, err = db.TransferLegacy(ctx, source, failed, true); err == nil {
		t.Fatal("failed import reported success")
	}
	for _, table := range []string{"users", "device", "analytics_outbox"} {
		if err = failed.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&count); err != nil || count != 0 {
			t.Fatalf("partial import leaked into %s: %d %v", table, count, err)
		}
	}
	if err = failed.QueryRow(ctx, "SELECT schema_version FROM app_metadata WHERE id=1").Scan(&bootstrapVersion); err != nil || bootstrapVersion != 1 {
		t.Fatalf("failed import did not restore bootstrap metadata: version=%d err=%v", bootstrapVersion, err)
	}
}
