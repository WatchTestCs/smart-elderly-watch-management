package care

import (
 "context"
 "fmt"
 "net/http"
 "net/http/httptest"
 "os"
 "strings"
 "testing"
 "time"

 "careold-go/internal/db"
 "github.com/google/uuid"
 "github.com/jackc/pgx/v5/pgxpool"
)

func testCareDB(t *testing.T)*pgxpool.Pool{
 t.Helper()
 dsn:=os.Getenv("TEST_DATABASE_URL")
 if dsn==""{t.Skip("PostgreSQL integration requires TEST_DATABASE_URL")}
 ctx:=context.Background()
 admin,err:=pgxpool.New(ctx,dsn);if err!=nil{t.Fatal(err)}
 schema:="test_"+strings.ReplaceAll(uuid.NewString(),"-","")
 if _,err=admin.Exec(ctx,"CREATE SCHEMA "+schema);err!=nil{admin.Close();t.Fatal(err)}
 cfg,err:=pgxpool.ParseConfig(dsn);if err!=nil{t.Fatal(err)}
 cfg.ConnConfig.RuntimeParams["search_path"]=schema
 pool,err:=pgxpool.NewWithConfig(ctx,cfg);if err!=nil{t.Fatal(err)}
 t.Cleanup(func(){pool.Close();admin.Exec(ctx,"DROP SCHEMA "+schema+" CASCADE");admin.Close()})
 if err:=db.Migrate(ctx,pool);err!=nil{t.Fatal(err)}
 for _,sql:=range []string{
  "INSERT INTO oldperson(old_id,name) VALUES(1,'长者')",
  "INSERT INTO device(device_id,device_no,secret,old_id) VALUES(1,'TEST001','unused',1)",
  "INSERT INTO care_incident(id,old_id,device_id,due_at) VALUES(1,1,1,now()+interval '5 minutes')",
 }{
  if _,err:=pool.Exec(ctx,sql);err!=nil{t.Fatal(err)}
 }
 return pool
}

func TestOutboxDispatchIsRecordedButNotFalselyDelivered(t *testing.T){
 pool:=testCareDB(t)
 ctx:=context.Background()
 if _,err:=pool.Exec(ctx,"INSERT INTO care_notification_outbox(incident_id,channel,recipient) VALUES(1,'sms','13800000001')");err!=nil{t.Fatal(err)}
 calls:=0
 server:=httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter,r *http.Request){
  calls++
  if r.Header.Get("Authorization")!="Bearer test-token"||r.Header.Get("Idempotency-Key")=="" {t.Error("missing gateway credentials or idempotency key")}
  w.WriteHeader(http.StatusAccepted)
 }))
 defer server.Close()
 d:=&dispatcher{db:pool,webhookURL:server.URL,webhookToken:"test-token",httpClient:&http.Client{Timeout:time.Second}}
 if err:=d.dispatchOne(ctx);err!=nil{t.Fatal(err)}
 if err:=d.dispatchOne(ctx);err!=nil{t.Fatal(err)}
 if calls!=1{t.Fatalf("dispatch count %d",calls)}
 var status,incident string
 if err:=pool.QueryRow(ctx,"SELECT status FROM care_notification_outbox").Scan(&status);err!=nil{t.Fatal(err)}
 if err:=pool.QueryRow(ctx,"SELECT state FROM care_incident WHERE id=1").Scan(&incident);err!=nil{t.Fatal(err)}
 if status!="sent"||incident!="received"{t.Fatalf("HTTP 202 is not delivery: outbox=%s incident=%s",status,incident)}
}

func TestOverdueSOSCreatesEscalationAuditAndFallback(t *testing.T){
 pool:=testCareDB(t)
 ctx:=context.Background()
 if _,err:=pool.Exec(ctx,"UPDATE care_incident SET due_at=now()-interval '1 minute' WHERE id=1");err!=nil{t.Fatal(err)}
 d:=&dispatcher{db:pool,escalationPhone:"13800000002"}
 if err:=d.escalate(ctx);err!=nil{t.Fatal(err)}
 if err:=d.escalate(ctx);err!=nil{t.Fatal(err)}
 var state string
 var audits,notices int
 if err:=pool.QueryRow(ctx,"SELECT state FROM care_incident WHERE id=1").Scan(&state);err!=nil{t.Fatal(err)}
 if err:=pool.QueryRow(ctx,"SELECT count(*) FROM care_incident_event WHERE action='escalate'").Scan(&audits);err!=nil{t.Fatal(err)}
 if err:=pool.QueryRow(ctx,"SELECT count(*) FROM care_notification_outbox").Scan(&notices);err!=nil{t.Fatal(err)}
 if state!="escalated"||audits!=1||notices!=1{t.Fatal(fmt.Sprintf("state=%s audits=%d notices=%d",state,audits,notices))}
}
