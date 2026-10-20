package handlers

import (
	dbstore "careold-go/internal/db"
	"careold-go/internal/db/dbtest"
	"context"
	"encoding/json"

	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"careold-go/internal/auth"
	"careold-go/internal/config"
	"careold-go/internal/csrf"
	"careold-go/internal/db"

	"careold-go/internal/session"
	"github.com/google/uuid"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Each test uses a separate temporary schema in a dedicated test database.
func testDatabase(t *testing.T) *dbstore.Database {
	if os.Getenv("TEST_MYSQL_DSN") != "" {
		return dbtest.MySQL(t)
	}
	t.Helper()
	dsn := os.Getenv("TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("TEST_DATABASE_URL not set; PostgreSQL integration test skipped")
	}
	ctx := context.Background()
	admin, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	schema := "test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, "CREATE SCHEMA "+schema); err != nil {
		admin.Close()
		t.Fatal(err)
	}
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatal(err)
	}
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	cfg.MaxConns = 8
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		pool.Close()
		_, err := admin.Exec(ctx, "DROP SCHEMA "+schema+" CASCADE")
		admin.Close()
		if err != nil {
			t.Error(err)
		}
	})
	if err = db.Migrate(ctx, pool); err != nil {
		t.Fatal(err)
	}
	return dbstore.Postgres(pool)
}

func api(t *testing.T, handler http.HandlerFunc, body any) map[string]any {
	t.Helper()
	response := callAPI(handler, body)
	if response.Code != 200 {
		t.Fatalf("HTTP %d: %s", response.Code, response.Body.String())
	}
	var out map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func callAPI(handler http.HandlerFunc, body any) *httptest.ResponseRecorder {
	data, err := json.Marshal(body)
	if err != nil {
		panic(err)
	}
	request := httptest.NewRequest("POST", "/restful/test", strings.NewReader(string(data)))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func testHandler(pool *dbstore.Database) *Handler {
	return New(pool, session.NewStore(time.Hour), config.Config{})
}

func mustExec(t *testing.T, pool *dbstore.Database, sql string, args ...any) {
	t.Helper()
	if _, err := pool.Exec(context.Background(), sql, args...); err != nil {
		t.Fatal(err)
	}
}

func TestIntegrationMigrationsAndBindingConstraint(t *testing.T) {
	pool := testDatabase(t)
	ctx := context.Background()
	// Applied migrations can be rerun concurrently without duplicate tracking rows.
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); errs <- migrateTestDB(ctx, pool) }()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM schema_migrations").Scan(&count); err != nil || count != expectedMigrations(pool) {
		t.Fatalf("migrations=%d err=%v", count, err)
	}
	_, err := pool.Exec(ctx, "INSERT INTO device(device_no,secret,old_id) VALUES('orphan','unused',999)")

	if !db.IsForeignKey(err) {
		t.Fatalf("orphan was not rejected: %v", err)
	}
	mustExec(t, pool, "INSERT INTO oldperson(old_id,name) VALUES(1,'长者')")
	mustExec(t, pool, "INSERT INTO device(device_no,secret,old_id) VALUES('bound','unused',1)")
	out := callAPI(testHandler(pool).oldPersonDelete, map[string]any{"oldId": 1})
	if out.Code != 409 {
		t.Fatalf("bound elder deletion: HTTP %d %s", out.Code, out.Body.String())
	}
	mustExec(t, pool, "UPDATE device SET old_id=NULL WHERE device_no='bound'")
	result := api(t, testHandler(pool).oldPersonDelete, map[string]any{"oldId": 1})
	if result["returnCode"] != "1111" {
		t.Fatal(result)
	}
}

func TestIntegrationListContractsAndNumericAge(t *testing.T) {
	pool := testDatabase(t)
	h := testHandler(pool)
	mustExec(t, pool, "INSERT INTO customer(customer_id,customer_name,customer_age,customer_sex,customer_phone,customer_address) VALUES(1,'监护人',40,'女','13800000001','本地')")
	mustExec(t, pool, "INSERT INTO family(family_id,family_name) VALUES(1,'家属')")
	mustExec(t, pool, "INSERT INTO oldperson(old_id,name,family_id) VALUES(1,'长者',1)")
	for _, test := range []struct {
		name    string
		handler http.HandlerFunc
		key     string
	}{
		{"customers", h.customerSelect, "customers"}, {"family", h.familySelect, "familyDtos"},
	} {
		t.Run(test.name, func(t *testing.T) {
			out := api(t, test.handler, map[string]any{"page": 1, "pageSize": 20})
			items, ok := out[test.key].([]any)
			if !ok || len(items) != 1 || out["total"] != float64(1) {
				t.Fatalf("incompatible list: %#v", out)
			}
			if test.key == "familyDtos" && items[0].(map[string]any)["name"] != "长者" {
				t.Fatal(out)
			}
		})
	}
	result := api(t, h.customerUpdate, map[string]any{"customerId": 1, "customerName": "监护人", "customerAge": 40, "customerSex": "女", "customerPhone": "13800000001", "customerAddress": "本地"})
	if result["returnCode"] != "1111" {
		t.Fatal(result)
	}
}

func TestIntegrationEditPreservesFamily(t *testing.T) {
	pool := testDatabase(t)
	h := testHandler(pool)
	mustExec(t, pool, "INSERT INTO oldperson(old_id,name,family_id) VALUES(1,'长者',7)")
	out := api(t, h.oldPersonUpdate, map[string]any{"oldId": 1, "name": "新姓名", "bothdate": "1950-01-01", "sex": "女", "phone": "13800000000", "room": "101", "illness": "", "warn": "", "foodLike": "", "foodHate": "", "drag": "", "customerName": "1"})
	if out["returnCode"] != "1111" {
		t.Fatal(out)
	}
	var familyID *int64
	if err := pool.QueryRow(context.Background(), "SELECT family_id FROM oldperson WHERE old_id=1").Scan(&familyID); err != nil || familyID == nil || *familyID != 7 {
		t.Fatalf("binding changed: %v err=%v", familyID, err)
	}
}

const testSecret = "test-device-secret-1234567890"

func seedDevice(t *testing.T, pool *dbstore.Database) {
	t.Helper()
	hash, err := secretHash(testSecret)
	if err != nil {
		t.Fatal(err)
	}
	mustExec(t, pool, "INSERT INTO oldperson(old_id,name) VALUES(1,'长者')")
	mustExec(t, pool, "INSERT INTO device(device_id,device_no,secret,old_id) VALUES(1,'WATCH001',$1,1)", hash)
}

func TestIntegrationCoordinatesAndSOS(t *testing.T) {
	pool := testDatabase(t)
	h := testHandler(pool)
	seedDevice(t, pool)
	// No current or historical location must still create an actionable SOS.
	out := api(t, h.deviceSos, map[string]any{"deviceNo": "WATCH001", "secret": testSecret})
	if out["returnCode"] != "1111" {
		t.Fatal(out)
	}
	var lng, lat *string
	if err := pool.QueryRow(context.Background(), "SELECT lng,lat FROM alarm_precess ORDER BY id DESC LIMIT 1").Scan(&lng, &lat); err != nil || lng != nil || lat != nil {
		t.Fatalf("missing position must be NULL: %v %v err=%v", lng, lat, err)
	}
	// Real handlers must accept numeric decimal coordinates for reports and SOS.
	for _, handler := range []http.HandlerFunc{h.deviceReport, h.deviceSos} {
		out = api(t, handler, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "lng": 121.4737, "lat": 31.2304, "hr": 75, "step": 120})
		if out["returnCode"] != "1111" {
			t.Fatal(out)
		}
	}
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM device_location WHERE lng='121.4737' AND lat='31.2304'").Scan(&count); err != nil || count != 2 {
		t.Fatalf("numeric coordinates persisted=%d err=%v", count, err)
	}
	bad := callAPI(h.deviceReport, map[string]any{"deviceNo": "WATCH001", "secret": testSecret, "lng": 181, "lat": 31.2})
	if bad.Code != 400 {
		t.Fatalf("invalid coordinates accepted: %d", bad.Code)
	}
}

func TestIntegrationDistinctInvitesCannotOverwriteFamily(t *testing.T) {
	pool := testDatabase(t)
	h := testHandler(pool)
	ctx := context.Background()
	mustExec(t, pool, "INSERT INTO oldperson(old_id,name) VALUES(1,'长者')")
	mustExec(t, pool, "INSERT INTO invite_code(code,old_id,expire_time) VALUES('AAAAAA',1,now()+interval '1 day'),('BBBBBB',1,now()+interval '1 day')")
	// Hold the parent row so both registrations contend on the same elder.
	blocker, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer blocker.Rollback(ctx)
	if _, err = blocker.Exec(ctx, "SELECT old_id FROM oldperson WHERE old_id=1 FOR UPDATE"); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan *httptest.ResponseRecorder, 2)
	for i, code := range []string{"AAAAAA", "BBBBBB"} {
		go func(i int, code string) {
			<-start
			results <- callAPI(h.inviteJoin, map[string]any{"code": code, "familyName": "家属", "familyPhone": fmt.Sprintf("1380000000%d", i+1), "familyPassword": "TestPassword123!", "familyRelation": "子女", "familySex": "女", "familyAddress": "本地"})
		}(i, code)
	}
	close(start)
	// Wait until both HTTP transactions have reached database lock contention.
	deadline := time.Now().Add(10 * time.Second)
	for {
		var waiting int
		if err = pool.QueryRow(ctx, lockWaitQuery(pool, false)).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting >= 2 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("registrations never reached row-lock contention")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err = blocker.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	success := 0
	for range 2 {
		res := <-results
		var out map[string]any
		if err = json.Unmarshal(res.Body.Bytes(), &out); err != nil {
			t.Fatal(err)
		}
		if out["returnCode"] == "1111" {
			success++
		}
	}
	var families, used int
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM family").Scan(&families); err != nil {
		t.Fatal(err)
	}
	if err = pool.QueryRow(ctx, "SELECT count(*) FROM invite_code WHERE used=1").Scan(&used); err != nil {
		t.Fatal(err)
	}
	if success != 1 || families != 1 || used != 1 {
		t.Fatalf("success=%d accounts=%d used=%d", success, families, used)
	}
	var bound bool
	if err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM oldperson o JOIN family f ON f.family_id=o.family_id WHERE o.old_id=1)").Scan(&bound); err != nil || !bound {
		t.Fatalf("missing winning binding: %v", err)
	}
}

func TestLogoutDestroysSessionAndRetainsCSRF(t *testing.T) {
	for _, role := range []string{auth.RoleFamily, auth.RoleAdmin} {
		t.Run(role, func(t *testing.T) {
			store := session.NewStore(time.Hour)
			h := New(nil, store, config.Config{})
			handler := csrf.Middleware(store, auth.Middleware(store, http.HandlerFunc(h.logout)))
			sess := store.Create(session.Principal{Role: role, ID: 1})
			request := func(token string) *httptest.ResponseRecorder {
				r := httptest.NewRequest("POST", "/restful/userLogin/logout", nil)
				r.AddCookie(&http.Cookie{Name: session.CookieName, Value: sess.ID})
				r.Header.Set(csrf.HeaderName, token)
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, r)
				return w
			}
			if res := request(""); res.Code != 403 || store.Get(sess.ID) == nil {
				t.Fatal("logout bypassed CSRF or destroyed session on rejection")
			}
			if res := request(sess.CSRFToken); res.Code != 200 || store.Get(sess.ID) != nil {
				t.Fatalf("logout failed: %d %s", res.Code, res.Body.String())
			}
		})
	}
}

func TestIntegrationBindDeleteRace(t *testing.T) {
	pool := testDatabase(t)
	h := testHandler(pool)
	ctx := context.Background()
	mustExec(t, pool, "INSERT INTO oldperson(old_id,name) VALUES(1,'长者')")
	mustExec(t, pool, "INSERT INTO device(device_no,secret) VALUES('racing','unused')")
	binding, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer binding.Rollback(ctx)
	if _, err = binding.Exec(ctx, "UPDATE device SET old_id=1 WHERE device_no='racing'"); err != nil {
		t.Fatal(err)
	}
	result := make(chan *httptest.ResponseRecorder, 1)
	go func() { result <- callAPI(h.oldPersonDelete, map[string]any{"oldId": 1}) }()
	deadline := time.Now().Add(10 * time.Second)
	for {
		var waiting int
		if err = pool.QueryRow(ctx, lockWaitQuery(pool, true)).Scan(&waiting); err != nil {
			t.Fatal(err)
		}
		if waiting > 0 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("delete did not reach FK contention")
		}
		time.Sleep(10 * time.Millisecond)
	}
	if err = binding.Commit(ctx); err != nil {
		t.Fatal(err)
	}
	response := <-result
	if response.Code != 409 {
		t.Fatalf("racing deletion: HTTP %d %s", response.Code, response.Body.String())
	}
	var intact bool
	if err = pool.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM device d JOIN oldperson o ON o.old_id=d.old_id WHERE d.device_no='racing')").Scan(&intact); err != nil || !intact {
		t.Fatalf("binding became orphaned: %v", err)
	}
}

func TestNativeSetupRequiresDesktopToken(t *testing.T) {
	pool := testDatabase(t)
	h := testHandler(pool)
	h.Cfg.DesktopToken = "test-only-bootstrap-token"
	mux := http.NewServeMux()
	h.routeAuth(mux)
	handler := csrf.Middleware(h.Sessions, auth.Middleware(h.Sessions, mux))
	for _, test := range []struct {
		name, token string
		status      int
	}{
		{"missing", "", 403}, {"wrong", "wrong-token", 403}, {"native", h.Cfg.DesktopToken, 200},
	} {
		t.Run(test.name, func(t *testing.T) {
			body := `{"name":"管理员","phone":"13800000000","password":"TestPassword123!"}`
			request := httptest.NewRequest("POST", "/restful/setup", strings.NewReader(body))
			request.Header.Set("X-Suining-Desktop", test.token)
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("HTTP %d %s", response.Code, response.Body.String())
			}
		})
	}
	var count int
	if err := pool.QueryRow(context.Background(), "SELECT count(*) FROM users").Scan(&count); err != nil || count != 1 {
		t.Fatalf("administrators=%d err=%v", count, err)
	}
	request := httptest.NewRequest("POST", "/restful/userLogin/login", strings.NewReader(`{}`))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != 403 || !strings.Contains(response.Body.String(), "CSRF_REJECTED") {
		t.Fatal("ordinary browser writes lost CSRF protection")
	}
}

func migrateTestDB(ctx context.Context, pool *db.Database) error {
	if pool.IsMySQL() {
		return db.MigrateMySQL(ctx, pool)
	}
	return db.Migrate(ctx, pool.LegacyPostgres())
}
func expectedMigrations(pool *db.Database) int {
	if pool.IsMySQL() {
		return 6
	}
	return 8
}

func lockWaitQuery(pool *dbstore.Database, deleting bool) string {
	if pool.IsMySQL() {
		return "SELECT count(*) FROM performance_schema.data_lock_waits w JOIN performance_schema.data_locks l ON l.ENGINE_LOCK_ID=w.REQUESTING_ENGINE_LOCK_ID AND l.ENGINE=w.ENGINE WHERE l.OBJECT_SCHEMA=DATABASE() AND l.OBJECT_NAME='oldperson'"
	}
	pattern := "%oldperson%"
	if deleting {
		pattern = "DELETE FROM oldperson%"
	}
	return "SELECT count(*) FROM pg_stat_activity WHERE datname=current_database() AND wait_event_type='Lock' AND query LIKE '" + pattern + "' AND pid<>pg_backend_pid()"
}
